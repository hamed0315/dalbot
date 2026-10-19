package handlers

import (
	"time"

	"dalbot/api"
	"dalbot/logger"
)

func handlePin(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	if replyToID == 0 {
		api.SendMessage(chatID, "❌ برای پین کردن، باید روی پیام مورد نظر ریپلای کنید.")
		return
	}

	if err := api.PinChatMessage(chatID, replyToID); err != nil {
		logger.Error.Printf("handlePin error: %v", err)
		api.SendMessage(chatID, "❌ خطا در پین کردن پیام. مطمئن شوید ربات ادمین است و دسترسی پین را دارد.")
		return
	}

	chatIDInt := parseInt64Chat(chatID)
	msg, _ := api.SendMessage(chatID, "📌 پیام پین شد.")
	logOperation(chatIDInt, "پین", true, "")
	if msg != nil {
		scheduleAutoDelete(chatIDInt, msg.MessageID, 10*time.Second)
	}
}

func handleUnpin(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	var msgID int
	if replyToID != 0 {
		msgID = replyToID
	}

	if err := api.UnpinChatMessage(chatID, msgID); err != nil {
		logger.Error.Printf("handleUnpin error: %v", err)
		api.SendMessage(chatID, "❌ خطا در حذف پین.")
		return
	}

	chatIDInt := parseInt64Chat(chatID)
	confirm, _ := api.SendMessage(chatID, "📍 پین حذف شد.")
	logOperation(chatIDInt, "حذف پین", true, "")
	if confirm != nil {
		scheduleAutoDelete(chatIDInt, confirm.MessageID, 10*time.Second)
	}
}
