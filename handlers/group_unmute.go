package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
)

// lookupUserByUsernameEnhanced ابتدا در mutes جستجو می‌کند، سپس private_users و group_users
func lookupUserByUsernameEnhanced(chatID int64, username string) (int, string, error) {
	var id int
	var name string
	// ۱. جستجو در mutes (کاربران سکوت‌شده)
	err := database.DB.QueryRow("SELECT user_id, first_name FROM mutes WHERE chat_id = ? AND username = ?", chatID, username).Scan(&id, &name)
	if err == nil {
		logger.Info.Printf("lookupUserByUsernameEnhanced: user %s found in mutes (id=%d)", username, id)
		return id, name, nil
	}
	// ۲. private_users
	err = database.DB.QueryRow("SELECT user_id, first_name FROM private_users WHERE username = ?", username).Scan(&id, &name)
	if err == nil {
		logger.Info.Printf("lookupUserByUsernameEnhanced: user %s found in private_users (id=%d)", username, id)
		return id, name, nil
	}
	// ۳. group_users
	err = database.DB.QueryRow("SELECT user_id, first_name FROM group_users WHERE chat_id = ? AND username = ?", chatID, username).Scan(&id, &name)
	if err == nil {
		logger.Info.Printf("lookupUserByUsernameEnhanced: user %s found in group_users (id=%d)", username, id)
		return id, name, nil
	}
	logger.Info.Printf("lookupUserByUsernameEnhanced: user %s not found in any table", username)
	return 0, "", fmt.Errorf("not found")
}

func handleUnmute(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	var resolvedID int
	var targetName string

	if targetUserID != 0 {
		resolvedID = targetUserID
		targetName = targetFirstName
	} else if len(args) > 0 {
		arg := args[0]
		if strings.HasPrefix(arg, "@") {
			chatIDInt := parseInt64Chat(chatID)
			foundID, foundName, err := lookupUserByUsernameEnhanced(chatIDInt, arg[1:])
			if err != nil {
				sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("❌ کاربری با یوزرنیم '%s' در لیست سکوت‌ها یا کاربران ربات یافت نشد. لطفاً از شناسه عددی استفاده کنید.", arg))
				return
			}
			resolvedID = foundID
			if foundName != "" {
				targetName = foundName
			} else {
				targetName = fmt.Sprintf("کاربر %d", resolvedID)
			}
		} else {
			id, err := strconv.Atoi(arg)
			if err != nil || id <= 0 {
				sendGroupMsg(chatID, AutoDel5, "❌ شناسه کاربر نامعتبر است.")
				return
			}
			resolvedID = id
			// سعی می‌کنیم نام را از mutes یا private_users پیدا کنیم
			row := database.DB.QueryRow("SELECT first_name FROM mutes WHERE chat_id = ? AND user_id = ?", parseInt64Chat(chatID), resolvedID)
			if err := row.Scan(&targetName); err != nil || targetName == "" {
				row = database.DB.QueryRow("SELECT first_name FROM private_users WHERE user_id = ?", resolvedID)
				if err := row.Scan(&targetName); err != nil || targetName == "" {
					targetName = fmt.Sprintf("کاربر %d", resolvedID)
				}
			}
		}
	} else {
		sendGroupMsg(chatID, AutoDel5, "❌ برای رفع سکوت، یکی از این راه‌ها:\n• ریپلای روی پیام کاربر\n• شناسه عددی: `رفع سکوت 123456`\n• یوزرنیم: `رفع سکوت @username`")
		return
	}

	if err := api.UnrestrictChatMember(chatID, resolvedID); err != nil {
		logger.Error.Printf("handleUnmute unrestrict error: %v", err)
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در رفع سکوت کاربر. ممکن است کاربر قبلاً آزاد شده باشد.")
		return
	}

	chatIDInt := parseInt64Chat(chatID)
	database.DB.Exec("DELETE FROM mutes WHERE chat_id = ? AND user_id = ?", chatIDInt, resolvedID)

	sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("🔊 سکوت %s توسط %s لغو شد.",
		"["+targetName+"](uid:"+strconv.Itoa(resolvedID)+")",
		"["+adminFirstName+"](uid:"+strconv.Itoa(userID)+")"))
	logOperation(chatIDInt, "رفع سکوت", true, fmt.Sprintf("کاربر %d", resolvedID))
}
