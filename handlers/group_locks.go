package handlers

import (
	"fmt"

	"dalbot/database"
	"dalbot/services"
)

// translateLock converts Persian lock names to internal English codes.
func translateLock(persianName string) (string, bool) {
	switch persianName {
	case "لینک":
		return "link", true
	case "فوروارد":
		return "forward", true
	case "تبلیغات":
		return "ad", true
	case "عکس":
		return "photo", true
	case "ویدئو", "ویدیو":
		return "video", true
	case "صوت":
		return "audio", true
	case "طولانی":
		return "longtext", true
	case "استیکر":
		return "sticker", true
	case "گیف":
		return "gif", true
	case "گروه":
		return "all", true
	case "نظرسنجی":
		return "poll", true
	default:
		return "", false
	}
}

func handleLock(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	if len(args) == 0 {
		sendGroupMsg(chatID, AutoDel5, "❌ لطفاً نوع قفل را مشخص کنید (لینک، فوروارد، تبلیغات، عکس، ویدئو، صوت، طولانی، استیکر، گیف، گروه، نظرسنجی).")
		return
	}

	lockCode, ok := translateLock(args[0])
	if !ok {
		sendGroupMsg(chatID, AutoDel5, "❌ نوع قفل نامعتبر است.")
		return
	}

	chatIDInt := parseInt64Chat(chatID)
	_, err := database.DB.Exec("INSERT OR REPLACE INTO locks (chat_id, lock_type, is_locked) VALUES (?, ?, 1)", chatIDInt, lockCode)
	if err != nil {
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در ثبت قفل.")
		return
	}
	// به‌روزرسانی کش قفل‌ها
	services.InvalidateLocksCache(chatIDInt)
	sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("🔒 قفل %s فعال شد.", args[0]))
}

func handleUnlock(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	if len(args) == 0 {
		sendGroupMsg(chatID, AutoDel5, "❌ لطفاً نوع قفل را مشخص کنید.")
		return
	}

	lockCode, ok := translateLock(args[0])
	if !ok {
		sendGroupMsg(chatID, AutoDel5, "❌ نوع قفل نامعتبر است.")
		return
	}

	chatIDInt := parseInt64Chat(chatID)
	_, err := database.DB.Exec("DELETE FROM locks WHERE chat_id = ? AND lock_type = ?", chatIDInt, lockCode)
	if err != nil {
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در باز کردن قفل.")
		return
	}
	// به‌روزرسانی کش قفل‌ها
	services.InvalidateLocksCache(chatIDInt)
	sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("🔓 قفل %s غیرفعال شد.", args[0]))
}
