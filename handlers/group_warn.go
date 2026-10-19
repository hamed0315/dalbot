package handlers

import (
	"fmt"
	"strconv"

	"dalbot/api"
)

func handleWarn(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	if targetUserID == 0 {
		sendGroupMsg(chatID, AutoDel5, "❌ برای اخطار دادن، باید روی پیام کاربر ریپلای کنید.")
		return
	}
	chatIDInt := parseInt64Chat(chatID)
	warnUser(chatIDInt, targetUserID, userID, "اخطار توسط ادمین")

	// 🆕 ساخت دکمهٔ مشاهده پروفایل
	profileBtn := api.InlineKeyboardButton{
		Text: fmt.Sprintf("👤 مشاهده پروفایل %s", targetFirstName),
		URL:  fmt.Sprintf("uid:%d", targetUserID),
	}
	keyboard := api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{{profileBtn}},
	}

	// ارسال پیام اخطار همراه با دکمه
	sentMsg, err := api.SendMessageWithKeyboard(chatID,
		fmt.Sprintf("⚠️ به کاربر %s اخطار داده شد.", "["+targetFirstName+"](uid:"+strconv.Itoa(targetUserID)+")"),
		keyboard,
	)
	if err == nil && sentMsg != nil {
		// زمان‌بندی حذف خودکار (۱۰ ثانیه)
		scheduleAutoDelete(chatIDInt, sentMsg.MessageID, AutoDel5)
	}
}
