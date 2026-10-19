package handlers

import (
	"fmt"
	"strconv"
	"time"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
	"dalbot/models"
	"dalbot/services"
)

func warnUser(chatID int64, userID int, warnedBy int, reason string) {
	err := models.AddWarning(chatID, int64(userID), "", int64(warnedBy), "", reason)
	if err != nil {
		logger.Error.Printf("Failed to add warning: %v", err)
	}
	// به‌روزرسانی username و first_name (از private_users یا group_users)
	var uname, fname string
	row := database.DB.QueryRow("SELECT username, first_name FROM private_users WHERE user_id = ?", userID)
	if err := row.Scan(&uname, &fname); err != nil {
		row = database.DB.QueryRow("SELECT username, first_name FROM group_users WHERE user_id = ? AND chat_id = ?", userID, chatID)
		if err := row.Scan(&uname, &fname); err != nil {
			uname = ""
			fname = ""
		}
	}
	database.DB.Exec("UPDATE warnings SET username = ?, first_name = ? WHERE chat_id = ? AND user_id = ? AND id = (SELECT MAX(id) FROM warnings WHERE chat_id = ? AND user_id = ?)", uname, fname, chatID, userID, chatID, userID)

	warnCount, _ := models.GetWarningCount(chatID, int64(userID))
	if warnCount >= 3 {
		api.BanChatMember(strconv.FormatInt(chatID, 10), userID)
		models.ClearWarnings(chatID, int64(userID))
		api.SendMessage(strconv.FormatInt(chatID, 10), "کاربر به دلیل دریافت ۳ اخطار از گروه بن شد.")
	} else {
		msg := fmt.Sprintf("⚠️ اخطار %d/3", warnCount)
		sentMsg, err := api.SendMessage(strconv.FormatInt(chatID, 10), msg)
		if err == nil {
			scheduleAutoDelete(chatID, sentMsg.MessageID, 10*time.Second)
		}
	}
}

func scheduleAutoDelete(chatID int64, messageID int, delay time.Duration) {
	services.AddAutoDelete(chatID, messageID, delay)
}

func parseInt64Chat(s string) int64 {
	var id int64
	fmt.Sscanf(s, "%d", &id)
	return id
}

func recordBan(chatID, userID int64, targetFirstName string, adminID int64, adminFirstName string, reason string) {
	database.DB.Exec(
		"INSERT INTO ban_history (chat_id, user_id, target_first_name, admin_id, admin_first_name, reason) VALUES (?, ?, ?, ?, ?, ?)",
		chatID, userID, targetFirstName, adminID, adminFirstName, reason,
	)
}

func recordMute(chatID, userID int64, targetFirstName string, adminID int64, adminFirstName string, durationMinutes int) {
	database.DB.Exec(
		"INSERT INTO mute_history (chat_id, user_id, target_first_name, admin_id, admin_first_name, duration_minutes) VALUES (?, ?, ?, ?, ?, ?)",
		chatID, userID, targetFirstName, adminID, adminFirstName, durationMinutes,
	)
}

func savePrivateUser(user *api.User) {
	database.DB.Exec(
		"INSERT OR REPLACE INTO private_users (user_id, first_name, username) VALUES (?, ?, ?)",
		user.ID, user.FirstName, user.Username,
	)
}

func getActiveGroupCount() (int, error) {
	row := database.DB.QueryRow("SELECT COUNT(*) FROM groups WHERE is_active = 1")
	var count int
	err := row.Scan(&count)
	return count, err
}

func getPrivateUserCount() int {
	row := database.DB.QueryRow("SELECT COUNT(*) FROM private_users")
	var count int
	row.Scan(&count)
	return count
}

func getBioLikeCount(chatID, targetUserID int64) int {
	row := database.DB.QueryRow("SELECT COUNT(*) FROM bio_likes WHERE chat_id = ? AND target_user_id = ?", chatID, targetUserID)
	var cnt int
	row.Scan(&cnt)
	return cnt
}

func likeBio(chatIDStr string, likerUserID, targetUserID, messageID int) {
	chatIDInt := parseInt64Chat(chatIDStr)

	var exists int
	database.DB.QueryRow("SELECT COUNT(*) FROM bio_likes WHERE chat_id = ? AND target_user_id = ? AND liker_user_id = ?",
		chatIDInt, targetUserID, likerUserID).Scan(&exists)
	if exists > 0 {
		return
	}

	_, err := database.DB.Exec("INSERT INTO bio_likes (chat_id, target_user_id, liker_user_id) VALUES (?, ?, ?)",
		chatIDInt, targetUserID, likerUserID)
	if err != nil {
		return
	}

	newCount := getBioLikeCount(chatIDInt, int64(targetUserID))

	var likerName, targetName string
	row := database.DB.QueryRow("SELECT first_name FROM private_users WHERE user_id = ?", likerUserID)
	if err := row.Scan(&likerName); err != nil || likerName == "" {
		likerName = fmt.Sprintf("کاربر %d", likerUserID)
	}
	row = database.DB.QueryRow("SELECT first_name FROM private_users WHERE user_id = ?", targetUserID)
	if err := row.Scan(&targetName); err != nil || targetName == "" {
		targetName = fmt.Sprintf("کاربر %d", targetUserID)
	}

	var bioText string
	database.DB.QueryRow("SELECT bio_text FROM user_bios WHERE chat_id = ? AND user_id = ?", chatIDInt, targetUserID).Scan(&bioText)
	newText := fmt.Sprintf("📝 معرفی [%s](uid:%d):\n\n%s\n\n❤️ %d لایک", targetName, targetUserID, bioText, newCount)

	likeText := fmt.Sprintf("💜 لایک (%d)", newCount)
	keyboard := api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{{Text: likeText, CallbackData: "like_bio:" + strconv.Itoa(targetUserID)}},
		},
	}
	api.EditMessageText(chatIDStr, messageID, newText, &keyboard)

	notifyMsg := fmt.Sprintf(
		"💜 %s عزیز، شما توسط %s لایک شدید! (%d لایک)",
		"["+targetName+"](uid:"+strconv.Itoa(targetUserID)+")",
		"["+likerName+"](uid:"+strconv.Itoa(likerUserID)+")",
		newCount,
	)
	api.SendMessage(chatIDStr, notifyMsg)
}

func logOperation(chatID int64, operation string, success bool, details string) {
	group, _ := models.GetGroup(chatID)
	groupTitle := "گروه"
	if group != nil {
		groupTitle = group.Title
	}

	if success {
		logger.Debug.Printf("✅ عملیات %s در گروه \"%s\" با موفقیت انجام شد. %s", operation, groupTitle, details)
	} else {
		logger.Debug.Printf("❌ عملیات %s در گروه \"%s\" شکست خورد: %s", operation, groupTitle, details)
	}
}
