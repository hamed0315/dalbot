package services

import "dalbot/api"

// IsSpamPoll checks if a message matches the "anonymous poll spam" pattern.
func IsSpamPoll(msg *api.Message) bool {
	// ۱. اگر متن یا کپشن وجود داشته باشد، نظرسنجی اسپم نیست
	if msg.Text != "" || msg.Caption != "" {
		return false
	}
	// ۲. اگر reply_markup (دکمه) نداشته باشد، نظرسنجی نیست
	if msg.ReplyMarkup == nil || len(msg.ReplyMarkup.InlineKeyboard) == 0 {
		return false
	}
	// ۳. پیام بدون محتوای متنی ولی با دکمه = نظرسنجی ناشناس
	return true
}
