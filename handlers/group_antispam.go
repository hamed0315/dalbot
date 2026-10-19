package handlers

import (
	"fmt"
	"strconv"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
	"dalbot/services"
)

// ensureAntispamRow ایجاد یک رکورد پیش‌فرض در صورت عدم وجود
func ensureAntispamRow(chatID int64) {
	_, err := database.DB.Exec("INSERT OR IGNORE INTO antispam_settings (chat_id) VALUES (?)", chatID)
	if err != nil {
		logger.Error.Printf("ensureAntispamRow: %v", err)
	}
}

func handleAntispamFlood(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	if len(args) < 2 {
		api.SendMessage(chatID, "❌ مثال: تنظیم اسپم سیل ۴ ۳")
		return
	}
	maxMsgs, _ := strconv.Atoi(args[0])
	seconds, _ := strconv.Atoi(args[1])
	if maxMsgs <= 0 || seconds <= 0 {
		api.SendMessage(chatID, "❌ مقادیر باید بزرگتر از صفر باشند.")
		return
	}
	chatIDInt := parseInt64Chat(chatID)
	ensureAntispamRow(chatIDInt) // ← ایجاد سطر در صورت نیاز
	_, err := database.DB.Exec("UPDATE antispam_settings SET flood_max_msgs = ?, flood_seconds = ? WHERE chat_id = ?", maxMsgs, seconds, chatIDInt)
	if err != nil {
		api.SendMessage(chatID, "❌ خطا در ذخیره تنظیمات.")
		return
	}
	services.InvalidateConfig(chatIDInt)
	logger.Info.Printf("Flood updated for %d", chatIDInt)
	api.SendMessage(chatID, fmt.Sprintf("✅ سیل اسپم به %d پیام در %d ثانیه تنظیم شد.", maxMsgs, seconds))
}

func handleAntispamDuplicate(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	if len(args) < 1 {
		api.SendMessage(chatID, "❌ مثال: تنظیم اسپم تکرار ۲")
		return
	}
	maxDup, _ := strconv.Atoi(args[0])
	if maxDup <= 0 {
		api.SendMessage(chatID, "❌ مقدار باید بزرگتر از صفر باشد.")
		return
	}
	chatIDInt := parseInt64Chat(chatID)
	ensureAntispamRow(chatIDInt)
	_, err := database.DB.Exec("UPDATE antispam_settings SET duplicate_max = ? WHERE chat_id = ?", maxDup, chatIDInt)
	if err != nil {
		api.SendMessage(chatID, "❌ خطا در ذخیره تنظیمات.")
		return
	}
	services.InvalidateConfig(chatIDInt)
	logger.Info.Printf("Duplicate updated for %d", chatIDInt)
	api.SendMessage(chatID, fmt.Sprintf("✅ تکرار اسپم به %d بار تنظیم شد.", maxDup))
}

func handleAntispamMute(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	if len(args) < 1 {
		api.SendMessage(chatID, "❌ مثال: تنظیم اسپم سکوت ۳")
		return
	}
	hours, _ := strconv.Atoi(args[0])
	if hours <= 0 {
		api.SendMessage(chatID, "❌ مقدار باید بزرگتر از صفر باشد.")
		return
	}
	chatIDInt := parseInt64Chat(chatID)
	ensureAntispamRow(chatIDInt)
	_, err := database.DB.Exec("UPDATE antispam_settings SET punishment = 'mute', mute_duration_hours = ? WHERE chat_id = ?", hours, chatIDInt)
	if err != nil {
		api.SendMessage(chatID, "❌ خطا در ذخیره تنظیمات.")
		return
	}
	services.InvalidateConfig(chatIDInt)
	cfg := services.GetAntispamConfig(chatIDInt)
	logger.Info.Printf("Mute punishment updated for %d: %s", chatIDInt, cfg.Punishment)
	api.SendMessage(chatID, fmt.Sprintf("✅ جریمهٔ اسپم به سکوت %d ساعته تنظیم شد.", hours))
}

func handleAntispamBan(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	ensureAntispamRow(chatIDInt) // ← ابتدا سطر را تضمین کن
	_, err := database.DB.Exec("UPDATE antispam_settings SET punishment = 'ban' WHERE chat_id = ?", chatIDInt)
	if err != nil {
		api.SendMessage(chatID, "❌ خطا در ذخیره تنظیمات.")
		return
	}
	services.InvalidateConfig(chatIDInt)
	cfg := services.GetAntispamConfig(chatIDInt)
	logger.Info.Printf("Ban punishment updated for %d: %s", chatIDInt, cfg.Punishment)
	api.SendMessage(chatID, "✅ جریمهٔ اسپم به اخراج تغییر کرد.")
}
