package handlers

import (
	"fmt"
	"strings"
	"time"

	"dalbot/api"
	"dalbot/services"
)

func handleTagMenu(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	keyboard := api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{
				{Text: "👥 تگ همه", CallbackData: "tag:all"},
				{Text: "⚡ تگ فعالان", CallbackData: "tag:active"},
			},
		},
	}
	sentMsg, err := api.SendMessageWithKeyboard(chatID, "🎯 **گزینه مورد نظر را انتخاب کنید:**", keyboard)
	if err == nil && sentMsg != nil {
		chatIDInt := parseInt64Chat(chatID)
		scheduleAutoDelete(chatIDInt, sentMsg.MessageID, 30*time.Second)
	}
}


// sendTagMessages لیست کاربران را در پیام‌های چندگانه (در صورت نیاز) ارسال می‌کند
func sendTagMessages(chatID string, header string, users []struct {
	UserID    int
	Username  string
	FirstName string
}) {
	if len(users) == 0 {
		api.SendMessage(chatID, header+"\n⚠️ هیچ کاربری یافت نشد.")
		return
	}

	var sb strings.Builder
	sb.WriteString(header + "\n")
	charLimit := 3800 // حاشیه امن زیر ۴۰۹۶

	for _, u := range users {
		mention := formatMention(u.UserID, u.Username, u.FirstName) + " "
		if sb.Len()+len(mention) > charLimit {
			// ارسال قطعه فعلی و شروع قطعه جدید
			api.SendMessage(chatID, sb.String())
			sb.Reset()
		}
		sb.WriteString(mention)
	}
	// ارسال آخرین قطعه
	if sb.Len() > 0 {
		api.SendMessage(chatID, sb.String())
	}
}

func performTagAll(chatID string) {
	chatIDInt := parseInt64Chat(chatID)
	users := services.GetGroupUsers(chatIDInt)
	sendTagMessages(chatID, "🔔 **همه:**", users)
}

func performTagActive(chatID string, hours int) {
	chatIDInt := parseInt64Chat(chatID)
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	users := services.GetActiveGroupUsers(chatIDInt, since)
	sendTagMessages(chatID, fmt.Sprintf("⚡ **کاربران فعال در %d ساعت گذشته:**", hours), users)
}
