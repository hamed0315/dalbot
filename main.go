package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"os"
	"sort"
	"strings"
	"time"

	"dalbot/api"
	"dalbot/config"
	"dalbot/database"
	"dalbot/handlers"
	"dalbot/logger"
	"dalbot/services"
)

var msgQueue = make(chan *api.Update, 1000)
var currentDBPath string

func main() {
	logger.Init()
	config.Load()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "dalbot.db"
	}
	currentDBPath = dbPath
	database.Init(dbPath)

	resp, err := http.Get(config.BaseURL + "/getMe")
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var meResp struct {
			Ok     bool `json:"ok"`
			Result struct {
				ID int `json:"id"`
			} `json:"result"`
		}
		if json.Unmarshal(body, &meResp) == nil && meResp.Ok {
			config.BotID = meResp.Result.ID
		}
	}

	services.LoadRankingFromDB()
	services.LoadGroupUsersFromDB()
	services.WarmUpCaches()

	for i := 0; i < 40; i++ {
		go worker()
	}
	services.StartRateLimiter(25)

	go safeGo(watchdog)
	go safeGo(services.StartPeriodicCheck)
	go safeGo(startUnmuteWorker)
	go safeGo(resetDailyRankings)
	go safeGo(startPersistentSaver)
	go safeGo(services.StartAutoDeleteWorker)
	go safeGo(startDBBackupWorker)
	go services.StartPendingMessagesCleanupWorker()
	go safeGo(handlers.StartSessionCleaner)
	go safeGo(func() { services.InitLearner() })
	go services.StartMultiPhaseCleanupWorker() // 🆕 پاک‌سازی تاریخچه اسپم چندفازی
	go services.StartReputationCleanup()

	go safeGo(func() {
		var offset int
		var consecutiveErrors int
		const maxErrorsBeforeSkip = 10

		updates, err := api.GetUpdates(-1)
		if err == nil && len(updates) > 0 {
			offset = updates[len(updates)-1].UpdateID + 1
		}
		for {
			updates, err := api.GetUpdates(offset)
			if err != nil {
				if strings.Contains(err.Error(), "non-JSON") {
					logger.Info.Printf("getUpdates non-JSON: %v", err)
				} else {
					logger.Error.Printf("getUpdates error: %v", err)
				}
				consecutiveErrors++
				if consecutiveErrors >= maxErrorsBeforeSkip {
					logger.Info.Println("Too many consecutive errors, skipping old updates...")
					skipUpdates, skipErr := api.GetUpdates(-1)
					if skipErr == nil && len(skipUpdates) > 0 {
						offset = skipUpdates[len(skipUpdates)-1].UpdateID + 1
					}
					drainQueue()
					consecutiveErrors = 0
				}
				time.Sleep(5 * time.Second)
				continue
			}
			consecutiveErrors = 0
			for _, upd := range updates {
				offset = upd.UpdateID + 1
				updCopy := deepCopyUpdate(&upd)
				select {
				case msgQueue <- updCopy:
				default:
					logger.Error.Println("msgQueue is full, dropping update")
				}
			}
		}
	})

	logger.Info.Println("Bot started on Bale API (Testing)")
	select {}
}

func drainQueue() {
	for {
		select {
		case <-msgQueue:
		default:
			return
		}
	}
}

func safeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error.Printf("Goroutine panic: %v", r)
			}
		}()
		fn()
	}()
}

func deepCopyUpdate(src *api.Update) *api.Update {
	if src == nil {
		return nil
	}
	dst := *src
	if src.Message != nil {
		msgCopy := *src.Message
		if src.Message.ReplyTo != nil {
			replyCopy := *src.Message.ReplyTo
			msgCopy.ReplyTo = &replyCopy
		}
		dst.Message = &msgCopy
	}
	return &dst
}

func worker() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error.Printf("Worker panic: %v", r)
		}
	}()
	for upd := range msgQueue {
		services.AcquireToken()
		handlers.ProcessUpdate(upd)
	}
}

func watchdog() {
	for {
		time.Sleep(30 * time.Second)
		if len(msgQueue) >= 800 {
			api.SendMessage(config.OwnerID, "🚨 هشدار: صف پیام‌ها بیش از ۸۰٪ پر است!")
		}
	}
}

func startUnmuteWorker() {
	for {
		time.Sleep(30 * time.Second)
		rows, err := database.DB.Query("SELECT chat_id, user_id FROM mutes WHERE unmute_at <= datetime('now')")
		if err != nil {
			continue
		}
		for rows.Next() {
			var chatID int64
			var userID int64
			if err := rows.Scan(&chatID, &userID); err != nil {
				continue
			}
			api.UnrestrictChatMember(strconv.FormatInt(chatID, 10), int(userID))
			database.DB.Exec("DELETE FROM mutes WHERE chat_id = ? AND user_id = ?", chatID, userID)
		}
		rows.Close()
	}
}

func resetDailyRankings() {
	for {
		time.Sleep(10 * time.Minute)
		rows, err := database.DB.Query("SELECT chat_id, daily_reset_at FROM groups WHERE is_active = 1")
		if err != nil {
			continue
		}
		for rows.Next() {
			var chatID int64
			var resetAt string
			if err := rows.Scan(&chatID, &resetAt); err != nil {
				continue
			}
			t, err := time.Parse("2006-01-02 15:04:05", resetAt)
			if err != nil {
				continue
			}
			if time.Since(t) >= 24*time.Hour {
				services.ResetDailyRanking(chatID)
				database.DB.Exec("DELETE FROM user_messages WHERE chat_id = ?", chatID)
				database.DB.Exec("UPDATE groups SET daily_reset_at = datetime('now') WHERE chat_id = ?", chatID)
				logger.Info.Printf("Daily ranking reset for chat %d", chatID)
			}
		}
		rows.Close()
	}
}

func startPersistentSaver() {
	for {
		time.Sleep(1 * time.Minute)
		services.SaveRankingToDB()
		services.SaveGroupUsersToDB()
	}
}

func startDBBackupWorker() {
	logger.Info.Println("DB Backup worker started")
	backupDir := "/home/hamed/dalbot_backups"
	for {
		time.Sleep(6 * time.Hour)
		timestamp := time.Now().Format("20060102_150405")
		backupFile := backupDir + "/dalbot_test_" + timestamp + ".db"
		_ = os.MkdirAll(backupDir, 0755)
		src, err := os.Open(currentDBPath)
		if err != nil {
			logger.Error.Println("DB Backup: cannot open source:", err)
			services.LogToOwner("WARNING", "⚠️ پشتیبان‌گیری دیتابیس ناموفق: باز کردن فایل مبدأ ممکن نیست.")
			continue
		}
		dst, err := os.Create(backupFile)
		if err != nil {
			src.Close()
			logger.Error.Println("DB Backup: cannot create backup:", err)
			services.LogToOwner("WARNING", "⚠️ پشتیبان‌گیری دیتابیس ناموفق: ایجاد فایل مقصد ممکن نیست.")
			continue
		}
		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()
		if err != nil {
			logger.Error.Println("DB Backup: copy failed:", err)
			services.LogToOwner("WARNING", "⚠️ پشتیبان‌گیری دیتابیس ناموفق: کپی با خطا مواجه شد.")
			continue
		}
		files, _ := os.ReadDir(backupDir)
		var backups []string
		for _, f := range files {
			if !f.IsDir() && strings.HasPrefix(f.Name(), "dalbot_test_") {
				backups = append(backups, f.Name())
			}
		}
		if len(backups) > 5 {
			sort.Strings(backups)
			for _, old := range backups[:len(backups)-5] {
				os.Remove(backupDir + "/" + old)
			}
		}
		logger.Info.Printf("DB Backup: %s created", backupFile)
	}
}
