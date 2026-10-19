package handlers

import (
	"fmt"
	"strconv"
	"time"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
)

func handleMute(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)

	if targetUserID == 0 {
		sendGroupMsg(chatID, AutoDel5, "❌ برای سکوت کردن، باید روی پیام کاربر ریپلای کنید.")
		return
	}

	durationMinutes := 60
	if len(args) > 0 {
		if mins, err := strconv.Atoi(args[0]); err == nil && mins > 0 {
			durationMinutes = mins
		}
	}

	if err := api.RestrictChatMember(chatID, targetUserID, false); err != nil {
		logger.Error.Printf("handleMute restrict error: %v", err)
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در سکوت کردن کاربر.")
		return
	}

	api.DeleteMessage(chatID, replyToID)

	unmuteAt := time.Now().Add(time.Duration(durationMinutes) * time.Minute).UTC().Format("2006-01-02 15:04:05")

	// 📛 دریافت اطلاعات کاربر مستقیماً از API بله
	var targetUsername, targetName string
	cm, err := api.GetChatMember(chatID, targetUserID)
	if err == nil && cm != nil {
		targetUsername = cm.User.Username
		targetName = cm.User.FirstName
		if targetName == "" {
			targetName = targetFirstName
		}
	} else {
		// اگر API خطا داد (مثلاً کاربر نیست)، از اطلاعات موجود در پیام ریپلای استفاده می‌کنیم
		logger.Error.Printf("handleMute: GetChatMember failed, using fallback: %v", err)
		targetUsername = ""
		targetName = targetFirstName
	}

	// ذخیره در دیتابیس با اطلاعات کامل
	database.DB.Exec("INSERT OR REPLACE INTO mutes (chat_id, user_id, unmute_at, username, first_name) VALUES (?, ?, ?, ?, ?)",
		chatIDInt, targetUserID, unmuteAt, targetUsername, targetName)

	recordMute(chatIDInt, int64(targetUserID), targetFirstName, int64(userID), adminFirstName, durationMinutes)

	confirmText := fmt.Sprintf("🔇 کاربر %s توسط %s به مدت %d دقیقه سکوت شد.",
		"["+targetFirstName+"](uid:"+strconv.Itoa(targetUserID)+")",
		"["+adminFirstName+"](uid:"+strconv.Itoa(userID)+")",
		durationMinutes)
	sendGroupMsg(chatID, AutoDel5, confirmText)

	logOperation(chatIDInt, "سکوت", true, fmt.Sprintf("کاربر %d", targetUserID))
}

func handleClearMutes(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)

	rows, err := database.DB.Query("SELECT user_id FROM mutes WHERE chat_id = ?", chatIDInt)
	if err != nil {
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در خواندن لیست سکوت‌ها.")
		return
	}
	defer rows.Close()

	var freedCount int
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			continue
		}
		api.UnrestrictChatMember(chatID, int(uid))
		freedCount++
	}

	database.DB.Exec("DELETE FROM mutes WHERE chat_id = ?", chatIDInt)

	if freedCount == 0 {
		sendGroupMsg(chatID, AutoDel5, "🔊 هیچ کاربری در حالت سکوت نبود.")
	} else {
		sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("🔊 سکوت %d کاربر لغو شد.", freedCount))
		logOperation(chatIDInt, "رفع سکوت همه", true, fmt.Sprintf("%d کاربر", freedCount))
	}
}
