package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
)

func handleBan(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	if targetUserID == 0 {
		sendGroupMsg(chatID, AutoDel5, "❌ برای بن کردن، باید روی پیام کاربر ریپلای کنید.")
		return
	}

	api.DeleteMessage(chatID, replyToID)

	if err := api.BanChatMember(chatID, targetUserID); err != nil {
		logger.Error.Printf("handleBan error: %v", err)
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در بن کردن کاربر.")
		return
	}

	chatIDInt := parseInt64Chat(chatID)
	recordBan(chatIDInt, int64(targetUserID), targetFirstName, int64(userID), adminFirstName, "بن توسط ادمین")
	sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("⛔ کاربر %s توسط %s از گروه بن شد.",
		"["+targetFirstName+"](uid:"+strconv.Itoa(targetUserID)+")",
		"["+adminFirstName+"](uid:"+strconv.Itoa(userID)+")"))
	logOperation(chatIDInt, "بن", true, fmt.Sprintf("کاربر %d", targetUserID))
}

func handleUnban(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	// روش ۱: از طریق ریپلای (targetUserID قبلاً توسط ProcessUpdate پر شده)
	if targetUserID != 0 {
		performUnban(chatID, userID, targetUserID, adminFirstName)
		return
	}

	// روش ۲ و ۳: از طریق آرگومان‌ها
	if len(args) == 0 {
		sendGroupMsg(chatID, AutoDel5, "❌ برای انبن کردن، یکی از این راه‌ها را به کار ببرید:\n"+
			"• ریپلای روی پیام کاربر\n"+
			"• شناسه عددی: `انبن 123456789`\n"+
			"• یوزرنیم: `انبن @username`")
		return
	}

	arg := args[0]
	var resolvedID int

	if strings.HasPrefix(arg, "@") {
		// جستجوی یوزرنیم در private_users
		err := database.DB.QueryRow("SELECT user_id FROM private_users WHERE username = ?", arg[1:]).Scan(&resolvedID)
		if err != nil {
			sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("❌ کاربری با یوزرنیم '%s' یافت نشد. مطمئن شوید که کاربر ربات را استارت کرده باشد.", arg))
			return
		}
	} else {
		// فرض بر شناسه عددی
		id, err := strconv.Atoi(arg)
		if err != nil || id <= 0 {
			sendGroupMsg(chatID, AutoDel5, "❌ شناسه کاربر نامعتبر است. یک عدد صحیح وارد کنید.")
			return
		}
		resolvedID = id
	}

	performUnban(chatID, userID, resolvedID, adminFirstName)
}

// performUnban executes the actual unban and sends confirmation.
func performUnban(chatID string, adminID int, targetUserID int, adminFirstName string) {
	if err := api.UnbanChatMember(chatID, targetUserID); err != nil {
		logger.Error.Printf("handleUnban error: %v", err)
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در انبن کردن کاربر. ممکن است کاربر قبلاً بن نشده باشد یا ربات دسترسی کافی نداشته باشد.")
		return
	}

	chatIDInt := parseInt64Chat(chatID)

	// تلاش برای یافتن نام کاربر هدف از private_users
	var targetName string
	row := database.DB.QueryRow("SELECT first_name FROM private_users WHERE user_id = ?", targetUserID)
	if err := row.Scan(&targetName); err != nil || targetName == "" {
		targetName = fmt.Sprintf("کاربر %d", targetUserID)
	}

	sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("🔓 کاربر %s توسط %s از حالت بن خارج شد و می‌تواند با لینک دعوت بازگردد.",
		"["+targetName+"](uid:"+strconv.Itoa(targetUserID)+")",
		"["+adminFirstName+"](uid:"+strconv.Itoa(adminID)+")"))
	logOperation(chatIDInt, "انبن", true, fmt.Sprintf("کاربر %d", targetUserID))
}
