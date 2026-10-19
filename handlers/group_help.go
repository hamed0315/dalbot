package handlers

import (
	"time"

	"fmt"
	"dalbot/api"
)

func handleHelp(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	level := getUserLevel(chatIDInt, userID)

	if level == "regular" || level == "special" {
		return
	}

	text := fmt.Sprintf("✨ %s عزیز، به **راهنمای جامع دال** خوش آمدی!\n\n🎯 یکی از بخش‌های زیر را انتخاب کنید:", adminFirstName)

	keyboard := api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{
				{Text: "🛡️ قفل‌ها", CallbackData: "help:locks"},
				{Text: "👥 مدیریت کاربران", CallbackData: "help:users"},
			},
			{
				{Text: "⚙️ تنظیمات ضداسپم", CallbackData: "help:antispam"},
				{Text: "🔗 لینک و دعوت", CallbackData: "help:invite"},
			},
			{
				{Text: "📊 آمار و رنکینگ", CallbackData: "help:stats"},
				{Text: "🧠 هوش مصنوعی", CallbackData: "help:ai"},
			},
			{
				{Text: "📌 سایر دستورات", CallbackData: "help:other"},
			},
			{
				{Text: "❌ بستن", CallbackData: "help:close"},
			},
		},
	}

	sentMsg, err := api.SendMessageWithKeyboard(chatID, text, keyboard)
	if err == nil && sentMsg != nil {
		scheduleAutoDelete(chatIDInt, sentMsg.MessageID, 60*time.Second)
	}
}
