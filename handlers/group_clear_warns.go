package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"dalbot/database"
	"dalbot/models"
)

func handleClearWarns(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	var resolvedID int
	var targetName string

	if targetUserID != 0 {
		resolvedID = targetUserID
		targetName = targetFirstName
	} else if len(args) > 0 {
		arg := args[0]
		if strings.HasPrefix(arg, "@") {
			chatIDInt := parseInt64Chat(chatID)
			// جستجو در warnings (اخطارها)
			err := database.DB.QueryRow("SELECT user_id, first_name FROM warnings WHERE chat_id = ? AND username = ?", chatIDInt, arg[1:]).Scan(&resolvedID, &targetName)
			if err != nil {
				// ممکن است username در warnings ذخیره نشده باشد، به private_users و group_users سقوط کنیم
				foundID, foundName, err2 := lookupUserByUsernameEnhanced(chatIDInt, arg[1:])
				if err2 != nil {
					sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("❌ کاربری با یوزرنیم '%s' در لیست اخطارها یا کاربران ربات یافت نشد. لطفاً از شناسه عددی استفاده کنید.", arg))
					return
				}
				resolvedID = foundID
				targetName = foundName
			}
		} else {
			id, err := strconv.Atoi(arg)
			if err != nil || id <= 0 {
				sendGroupMsg(chatID, AutoDel5, "❌ شناسه کاربر نامعتبر است.")
				return
			}
			resolvedID = id
			// سعی برای پیدا کردن نام از private_users
			row := database.DB.QueryRow("SELECT first_name FROM private_users WHERE user_id = ?", resolvedID)
			if err := row.Scan(&targetName); err != nil || targetName == "" {
				targetName = fmt.Sprintf("کاربر %d", resolvedID)
			}
		}
	} else {
		sendGroupMsg(chatID, AutoDel5, "❌ برای پاک کردن اخطارها، یکی از این راه‌ها:\n• ریپلای روی پیام کاربر\n• شناسه عددی: `حذف اخطار 123456`\n• یوزرنیم: `حذف اخطار @username`")
		return
	}

	chatIDInt := parseInt64Chat(chatID)
	err := models.ClearWarnings(chatIDInt, int64(resolvedID))
	if err != nil {
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در پاک کردن اخطارها.")
		return
	}

	sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("🧹 تمام اخطارهای %s پاک شد.",
		"["+targetName+"](uid:"+strconv.Itoa(resolvedID)+")"))
}
