package services

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
)

// ==================== Volume‑based Auto‑Clean (Event‑Driven + TTL) ====================
var (
	pendingMessages      = make(map[int64][]int)
	pendingDuringClean   = make(map[int64][]int)   // بافر موقت هنگام پاکسازی
	chatLastMsgTime      = make(map[int64]time.Time)
	pmMutex              sync.Mutex
	cleaningLocks        = make(map[int64]bool)
	clMutex              sync.Mutex
)

// ==================== Purge‑specific queue (for manual /پاکسازی command) ====================
const maxPurgeQueueSize = 500

var (
	purgeQueues = make(map[int64][]int)
	pqMutex     sync.Mutex
)

// AddUserMessageToPurge adds a message ID to the dedicated purge queue.
func AddUserMessageToPurge(chatID int64, messageID int) {
	pqMutex.Lock()
	defer pqMutex.Unlock()

	q := purgeQueues[chatID]
	q = append(q, messageID)
	if len(q) > maxPurgeQueueSize {
		q = q[len(q)-maxPurgeQueueSize:]
	}
	purgeQueues[chatID] = q
}

// GetPurgeMessages returns up to 'limit' messages from the purge queue without removing them.
func GetPurgeMessages(chatID int64, limit int) []int {
	pqMutex.Lock()
	defer pqMutex.Unlock()

	q := purgeQueues[chatID]
	if len(q) == 0 {
		return nil
	}
	if limit > len(q) {
		limit = len(q)
	}
	// فقط خواندن، بدون حذف
	result := make([]int, limit)
	copy(result, q[:limit])
	return result
}

// RemovePurgeMessages removes specific message IDs from the purge queue after successful deletion.
func RemovePurgeMessages(chatID int64, messageIDs []int) {
	pqMutex.Lock()
	defer pqMutex.Unlock()

	q := purgeQueues[chatID]
	if len(q) == 0 {
		return
	}
	toRemove := make(map[int]bool, len(messageIDs))
	for _, id := range messageIDs {
		toRemove[id] = true
	}
	var newQ []int
	for _, id := range q {
		if !toRemove[id] {
			newQ = append(newQ, id)
		}
	}
	purgeQueues[chatID] = newQ
}

func AddPendingMessage(chatID int64, messageID int) {
	pmMutex.Lock()
	defer pmMutex.Unlock()

	// ذخیره در جدول purge_queue برای ماندگاری
	database.DB.Exec("INSERT INTO purge_queue (chat_id, message_id, created_at) VALUES (?, ?, ?)",
		chatID, messageID, time.Now().UTC().Format("2006-01-02 15:04:05"))

	// 🔥 اگر قفل پاکسازی فعال است، پیام را به بافر موقت اضافه کن
	clMutex.Lock()
	locked := cleaningLocks[chatID]
	clMutex.Unlock()
	if locked {
		pendingDuringClean[chatID] = append(pendingDuringClean[chatID], messageID)
		return
	}

	pendingMessages[chatID] = append(pendingMessages[chatID], messageID)
	chatLastMsgTime[chatID] = time.Now()

	if len(pendingMessages[chatID]) >= 500 {
		clMutex.Lock()
		if cleaningLocks[chatID] {
			clMutex.Unlock()
			return
		}
		cleaningLocks[chatID] = true
		clMutex.Unlock()

		chatMessages := pendingMessages[chatID]
		pendingMessages[chatID] = nil
		go cleanMessages(chatID, chatMessages)
	}
}

func RemovePendingMessage(chatID int64, messageID int) {
	pmMutex.Lock()
	defer pmMutex.Unlock()
	list := pendingMessages[chatID]
	for i, id := range list {
		if id == messageID {
			pendingMessages[chatID] = append(list[:i], list[i+1:]...)
			break
		}
	}
}

func cleanMessages(chatID int64, messageIDs []int) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error.Printf("cleanMessages panic: %v", r)
		}
		// 🔥 پس از اتمام پاکسازی، محتوای بافر موقت را به صف اصلی منتقل کن و قفل را بردار
		pmMutex.Lock()
		if buf := pendingDuringClean[chatID]; len(buf) > 0 {
			pendingMessages[chatID] = append(pendingMessages[chatID], buf...)
			delete(pendingDuringClean, chatID)
			// اگر با اضافه شدن بافر، صف دوباره پر شد، دوباره پاکسازی را زمان‌بندی کن
			if len(pendingMessages[chatID]) >= 500 {
				chatMessages := pendingMessages[chatID]
				pendingMessages[chatID] = nil
				pmMutex.Unlock()
				go cleanMessages(chatID, chatMessages)
				// قفل برداشته نمی‌شود چون پاکسازی جدید شروع شده
				return
			}
		}
		pmMutex.Unlock()

		clMutex.Lock()
		delete(cleaningLocks, chatID)
		clMutex.Unlock()
	}()

	deleted := 0
	for _, msgID := range messageIDs {
		AcquireToken()
		err := api.DeleteMessage(fmt.Sprintf("%d", chatID), msgID)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				logger.Debug.Printf("AutoClean 404 (already deleted): %d/%d", chatID, msgID)
				continue
			}
			if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "Too Many Requests") {
				time.Sleep(500 * time.Millisecond)
				AcquireToken()
				err = api.DeleteMessage(fmt.Sprintf("%d", chatID), msgID)
				if err == nil {
					deleted++
				} else {
					logger.Error.Printf("AutoClean retry failed for %d: %v", msgID, err)
				}
				continue
			}
			logger.Error.Printf("AutoClean delete error for %d: %v", msgID, err)
			continue
		}
		deleted++
	}
	if deleted > 0 {
		logger.Debug.Printf("AutoClean: %d messages in chat %d", deleted, chatID)
	}
}

func GetPendingMessages(chatID int64, limit int) []int {
	pmMutex.Lock()
	defer pmMutex.Unlock()
	list := pendingMessages[chatID]
	if len(list) == 0 {
		return nil
	}
	if limit > len(list) {
		limit = len(list)
	}
	result := list[:limit]
	pendingMessages[chatID] = list[limit:]
	return result
}

func CleanupOldPendingMessages() {
	pmMutex.Lock()
	defer pmMutex.Unlock()
	cutoff := time.Now().Add(-24 * time.Hour)
	for chatID, lastTime := range chatLastMsgTime {
		if lastTime.Before(cutoff) {
			delete(pendingMessages, chatID)
			delete(chatLastMsgTime, chatID)
		}
	}
}

func StartPendingMessagesCleanupWorker() {
	for {
		time.Sleep(10 * time.Minute)
		CleanupOldPendingMessages()
	}
}

// ==================== Ranking (user_messages) ====================
var rankingStore = struct {
	sync.Mutex
	data map[int64]map[int]int
}{data: make(map[int64]map[int]int)}

func IncrementMessageCount(chatID int64, userID int) {
	rankingStore.Lock()
	defer rankingStore.Unlock()
	if _, ok := rankingStore.data[chatID]; !ok {
		rankingStore.data[chatID] = make(map[int]int)
	}
	rankingStore.data[chatID][userID]++
}

func GetTopUsers(chatID int64, limit int) []struct {
	UserID int
	Count  int
} {
	rankingStore.Lock()
	defer rankingStore.Unlock()
	chatMap := rankingStore.data[chatID]
	if chatMap == nil {
		return nil
	}
	type entry struct {
		uid int
		cnt int
	}
	var entries []entry
	for uid, cnt := range chatMap {
		entries = append(entries, entry{uid, cnt})
	}
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].cnt > entries[i].cnt {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	if len(entries) > limit {
		entries = entries[:limit]
	}
	result := make([]struct {
		UserID int
		Count  int
	}, len(entries))
	for i, e := range entries {
		result[i].UserID = e.uid
		result[i].Count = e.cnt
	}
	return result
}

func ResetDailyRanking(chatID int64) {
	rankingStore.Lock()
	defer rankingStore.Unlock()
	delete(rankingStore.data, chatID)
}

func SaveRankingToDB() {
	rankingStore.Lock()
	defer rankingStore.Unlock()
	for chatID, chatMap := range rankingStore.data {
		for userID, cnt := range chatMap {
			database.DB.Exec(
				"INSERT OR REPLACE INTO user_messages (chat_id, user_id, message_count) VALUES (?, ?, ?)",
				chatID, userID, cnt,
			)
		}
	}
}

func LoadRankingFromDB() {
	rows, err := database.DB.Query("SELECT chat_id, user_id, message_count FROM user_messages")
	if err != nil {
		logger.Error.Printf("LoadRankingFromDB: %v", err)
		return
	}
	defer rows.Close()
	rankingStore.Lock()
	defer rankingStore.Unlock()
	for rows.Next() {
		var chatID int64
		var userID int
		var cnt int
		if err := rows.Scan(&chatID, &userID, &cnt); err != nil {
			continue
		}
		if _, ok := rankingStore.data[chatID]; !ok {
			rankingStore.data[chatID] = make(map[int]int)
		}
		rankingStore.data[chatID][userID] = cnt
	}
}

// ==================== Group Users (for tag) ====================
var groupUsersStore = struct {
	sync.Mutex
	data map[int64]map[int]struct {
		Username  string
		FirstName string
		LastActive time.Time
	}
}{data: make(map[int64]map[int]struct {
	Username  string
	FirstName string
	LastActive time.Time
})}

func RecordUserActivity(chatID int64, userID int, username, firstName string) {
	groupUsersStore.Lock()
	defer groupUsersStore.Unlock()
	if _, ok := groupUsersStore.data[chatID]; !ok {
		groupUsersStore.data[chatID] = make(map[int]struct {
			Username  string
			FirstName string
			LastActive time.Time
		})
	}
	groupUsersStore.data[chatID][userID] = struct {
		Username  string
		FirstName string
		LastActive time.Time
	}{username, firstName, time.Now()}
}

func GetGroupUsers(chatID int64) []struct {
	UserID    int
	Username  string
	FirstName string
} {
	groupUsersStore.Lock()
	defer groupUsersStore.Unlock()
	chatMap := groupUsersStore.data[chatID]
	if chatMap == nil {
		return nil
	}
	var users []struct {
		UserID    int
		Username  string
		FirstName string
	}
	for uid, info := range chatMap {
		users = append(users, struct {
			UserID    int
			Username  string
			FirstName string
		}{uid, info.Username, info.FirstName})
	}
	return users
}

func GetActiveGroupUsers(chatID int64, since time.Time) []struct {
	UserID    int
	Username  string
	FirstName string
} {
	groupUsersStore.Lock()
	defer groupUsersStore.Unlock()
	chatMap := groupUsersStore.data[chatID]
	if chatMap == nil {
		return nil
	}
	var users []struct {
		UserID    int
		Username  string
		FirstName string
	}
	for uid, info := range chatMap {
		if info.LastActive.After(since) {
			users = append(users, struct {
				UserID    int
				Username  string
				FirstName string
			}{uid, info.Username, info.FirstName})
		}
	}
	return users
}

func SaveGroupUsersToDB() {
	groupUsersStore.Lock()
	defer groupUsersStore.Unlock()
	for chatID, chatMap := range groupUsersStore.data {
		for userID, info := range chatMap {
			database.DB.Exec(
				"INSERT OR REPLACE INTO group_users (chat_id, user_id, username, first_name, last_active) VALUES (?, ?, ?, ?, ?)",
				chatID, userID, info.Username, info.FirstName, info.LastActive,
			)
		}
	}
}

func LoadGroupUsersFromDB() {
	rows, err := database.DB.Query("SELECT chat_id, user_id, username, first_name, last_active FROM group_users")
	if err != nil {
		return
	}
	defer rows.Close()
	groupUsersStore.Lock()
	defer groupUsersStore.Unlock()
	for rows.Next() {
		var chatID int64
		var userID int
		var uname, fname string
		var lastActive time.Time
		if err := rows.Scan(&chatID, &userID, &uname, &fname, &lastActive); err != nil {
			continue
		}
		if _, ok := groupUsersStore.data[chatID]; !ok {
			groupUsersStore.data[chatID] = make(map[int]struct {
				Username  string
				FirstName string
				LastActive time.Time
			})
		}
		groupUsersStore.data[chatID][userID] = struct {
			Username  string
			FirstName string
			LastActive time.Time
		}{uname, fname, lastActive}
	}
}
