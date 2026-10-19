package services

import (
	"fmt"
	"sync"
	"time"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
	"dalbot/models"
)

var lastCheckTime = make(map[int64]time.Time)
var checkMutex sync.Mutex

func StartPeriodicCheck() {
	go func() {
		for {
			time.Sleep(2 * time.Minute)
			rows, err := database.DB.Query("SELECT chat_id, title, owner_id, is_active FROM groups")
			if err != nil {
				logger.Error.Println("Periodic check query error:", err)
				continue
			}
			for rows.Next() {
				var chatID int64
				var title string
				var ownerID int64
				var isActive bool
				if err := rows.Scan(&chatID, &title, &ownerID, &isActive); err != nil {
					continue
				}

				ownerInChannels := IsUserInChannels(int(ownerID))

				if !isActive && ownerInChannels {
					models.SetGroupActive(chatID, true)
					msg := fmt.Sprintf("✅ مالک گروه \"%s\" به کانال‌های حمایتی بازگشت. ربات دوباره فعال شد.", title)
					api.SendMessage(fmt.Sprintf("%d", chatID), msg)
					LogToOwner("INFO", fmt.Sprintf("✅ گروه \"%s\" (ID: %d) دوباره فعال شد.", title, chatID))
				} else if isActive && !ownerInChannels {
					go func(chatID int64, title string) {
						time.Sleep(1 * time.Minute)
						if !IsUserInChannels(int(ownerID)) {
							models.SetGroupActive(chatID, false)
							msg := fmt.Sprintf(
								"⚠️ مالک گروه \"%s\" از کانال‌های حمایتی خارج شده است.\nربات تا بازگشت مالک غیرفعال می‌باشد.",
								title,
							)
							keyboard := api.InlineKeyboardMarkup{
								InlineKeyboard: [][]api.InlineKeyboardButton{
									{{Text: "🔹 عضویت در کانال ۱", URL: "https://ble.ir/nobino78"}},
									{{Text: "🔹 عضویت در کانال ۲", URL: "https://ble.ir/linkdoni78"}},
									{{Text: "🔹 عضویت در کانال ۳", URL: "https://ble.ir/xau_78"}},
								},
							}
							api.SendMessageWithKeyboard(fmt.Sprintf("%d", chatID), msg, keyboard)
							LogToOwner("CRITICAL", fmt.Sprintf("🚨 گروه \"%s\" (ID: %d) به دلیل خروج مالک از کانال حمایتی غیرفعال شد.", title, chatID))
						}
					}(chatID, title)
				}
			}
			rows.Close()
		}
	}()
}

func CheckAndUpdateGroupStatus(chatID int64, ownerID int64) {
	checkMutex.Lock()
	lastTime, exists := lastCheckTime[chatID]
	if exists && time.Since(lastTime) < 30*time.Second {
		checkMutex.Unlock()
		return
	}
	lastCheckTime[chatID] = time.Now()
	checkMutex.Unlock()

	ownerInChannels := IsUserInChannels(int(ownerID))
	group, err := models.GetGroup(chatID)
	if err != nil || group == nil {
		return
	}
	if !ownerInChannels && group.IsActive {
		models.SetGroupActive(chatID, false)
		title := group.Title
		if title == "" {
			title = fmt.Sprintf("%d", chatID)
		}
		msg := fmt.Sprintf(
			"⚠️ مالک گروه \"%s\" از کانال‌های حمایتی خارج شده است.\nربات تا بازگشت مالک غیرفعال می‌باشد.",
			title,
		)
		keyboard := api.InlineKeyboardMarkup{
			InlineKeyboard: [][]api.InlineKeyboardButton{
				{{Text: "🔹 عضویت در کانال ۱", URL: "https://ble.ir/nobino78"}},
				{{Text: "🔹 عضویت در کانال ۲", URL: "https://ble.ir/linkdoni78"}},
				{{Text: "🔹 عضویت در کانال ۳", URL: "https://ble.ir/xau_78"}},
			},
		}
		api.SendMessageWithKeyboard(fmt.Sprintf("%d", chatID), msg, keyboard)
		LogToOwner("CRITICAL", fmt.Sprintf("🚨 گروه \"%s\" (ID: %d) به دلیل خروج مالک از کانال حمایتی غیرفعال شد (CheckAndUpdate).", title, chatID))
	} else if ownerInChannels && !group.IsActive {
		models.SetGroupActive(chatID, true)
		title := group.Title
		if title == "" {
			title = fmt.Sprintf("%d", chatID)
		}
		msg := fmt.Sprintf("✅ مالک گروه \"%s\" به کانال‌های حمایتی بازگشت. ربات دوباره فعال شد.", title)
		api.SendMessage(fmt.Sprintf("%d", chatID), msg)
		LogToOwner("INFO", fmt.Sprintf("✅ گروه \"%s\" (ID: %d) دوباره فعال شد (CheckAndUpdate).", title, chatID))
	}
}
