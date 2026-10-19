package handlers

import (
	"dalbot/api"
)

func handleLink(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatInfo, err := api.GetChat(chatID)
	if err != nil {
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در دریافت لینک. مطمئن شوید ربات ادمین است.")
		return
	}
	if chatInfo.InviteLink == "" {
		sendGroupMsg(chatID, AutoDel5, "❌ این گروه لینک دعوت عمومی ندارد.")
		return
	}
	sendGroupMsg(chatID, AutoDel5, "🔗 لینک دعوت گروه:\n"+chatInfo.InviteLink)
}
