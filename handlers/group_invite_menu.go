package handlers

import (
	"fmt"
	"time"

	"dalbot/api"
	"dalbot/database"
)

// handleInviteMenu shows a menu with three options (admin only). Auto-deletes after 30s.
func handleInviteMenu(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	// 🛡️ فقط ادمین‌ها و بالاتر
	level := getUserLevel(parseInt64Chat(chatID), userID)
	if level != "super_owner" && level != "owner" && level != "senior_admin" && level != "admin" {
		api.SendMessage(chatID, "⛔ شما مجوز اجرای این دستور را ندارید.")
		return
	}

	keyboard := api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{
				{Text: "🔗 ساخت لینک دال", CallbackData: "invite_menu:smart"},
				{Text: "🔗 لینک فعلی هوشمند دال", CallbackData: "invite_menu:show_smart"},
			},
			{
				{Text: "🔗 لینک گروه", CallbackData: "invite_menu:normal"},
			},
		},
	}
	sentMsg, err := api.SendMessageWithKeyboard(chatID, "🎯 **نوع لینک دعوت را انتخاب کنید:**", keyboard)
	if err == nil && sentMsg != nil {
		chatIDInt := parseInt64Chat(chatID)
		scheduleAutoDelete(chatIDInt, sentMsg.MessageID, 30*time.Second)
	}
}

// handleShowSmartLink reads the existing smart invite link from DB and sends it (auto-delete after 10s).
func handleShowSmartLink(chatID string, userID int) {
	var code string
	err := database.DB.QueryRow("SELECT code FROM invite_links WHERE chat_id = ? ORDER BY created_at DESC LIMIT 1", parseInt64Chat(chatID)).Scan(&code)
	if err != nil {
		sendGroupMsg(chatID, AutoDel5, "❌ هنوز هیچ لینک هوشمندی برای این گروه ساخته نشده است. از گزینه «ساخت لینک دال» استفاده کنید.")
		return
	}
	link := fmt.Sprintf("https://ble.ir/daal78bot?start=%s", code)
	sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("🔗 لینک هوشمند فعلی:\n%s\n\n📌 این لینک را در لینکدونی‌ها قرار دهید.", link))
}
