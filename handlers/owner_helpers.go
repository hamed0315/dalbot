package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"dalbot/api"
	"dalbot/config"
	"dalbot/database"
	"dalbot/logger"
)

// ============================================================
//  پنل مالک (super_owner)
// ============================================================
func showOwnerPanel(chatID string) {
	text := "👑 **پنل مالک ربات**\n\n" +
		"📢 ارسال پیام به همه گروه‌ها\n" +
		"👥 ارسال پیام به همه کاربران\n" +
		"📊 آمار ربات"
	keyboard := api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{{Text: "📢 ارسال به گروه‌ها", CallbackData: "panel_broadcast_groups"}},
			{{Text: "👥 ارسال به کاربران", CallbackData: "panel_broadcast_users"}},
			{{Text: "📊 آمار", CallbackData: "panel_stats"}},
		},
	}
	api.SendMessageWithKeyboard(chatID, text, keyboard)
}

// ============================================================
//  مدیریت مکالمات چندمرحله‌ای
// ============================================================
func handleConversationState(userID int64, state, text string) {
	switch state {
	case "awaiting_broadcast_user":
		// ارسال پیام به همهٔ کاربران
		rows, err := database.DB.Query("SELECT user_id FROM private_users")
		if err != nil {
			logger.Error.Printf("broadcast users query error: %v", err)
			api.SendMessage(strconv.FormatInt(userID, 10), "❌ خطا در دریافت لیست کاربران.")
			delete(conversationState, userID)
			return
		}
		defer rows.Close()
		count := 0
		for rows.Next() {
			var uid int64
			if err := rows.Scan(&uid); err == nil {
				api.SendMessage(strconv.FormatInt(uid, 10), "📢 پیام از طرف مالک:\n\n"+text)
				count++
			}
		}
		api.SendMessage(strconv.FormatInt(userID, 10), fmt.Sprintf("✅ پیام به %d کاربر ارسال شد.", count))
		delete(conversationState, userID)

	case "awaiting_broadcast_group":
		// ارسال پیام به همهٔ گروه‌های فعال
		rows, err := database.DB.Query("SELECT chat_id FROM groups WHERE is_active = 1")
		if err != nil {
			logger.Error.Printf("broadcast groups query error: %v", err)
			api.SendMessage(strconv.FormatInt(userID, 10), "❌ خطا در دریافت لیست گروه‌ها.")
			delete(conversationState, userID)
			return
		}
		defer rows.Close()
		count := 0
		for rows.Next() {
			var chatID int64
			if err := rows.Scan(&chatID); err == nil {
				api.SendMessage(strconv.FormatInt(chatID, 10), "📢 پیام از طرف مالک:\n\n"+text)
				count++
			}
		}
		api.SendMessage(strconv.FormatInt(userID, 10), fmt.Sprintf("✅ پیام به %d گروه ارسال شد.", count))
		delete(conversationState, userID)

	default:
		delete(conversationState, userID)
		api.SendMessage(strconv.FormatInt(userID, 10), "⚠️ دستور نامعتبر است. لطفاً دوباره تلاش کنید.")
	}
}

// ============================================================
//  مدیریت تأیید/رد دعوت
// ============================================================
func handleApprovalCallback(query *api.CallbackQuery) {
	data := query.Data
	parts := splitCallbackData(data)
	if len(parts) < 3 {
		return
	}

	chatIDInt, _ := strconv.ParseInt(parts[1], 10, 64)
	userID, _ := strconv.Atoi(parts[2])

	if strings.HasPrefix(data, "approve_invite:") {
		// تأیید دعوت
		link, err := api.ExportChatInviteLink(strconv.FormatInt(chatIDInt, 10))
		if err != nil {
			api.AnswerCallback(query.ID, "❌ خطا در ایجاد لینک دعوت.", true)
			return
		}
		// ارسال لینک به کاربر
		api.SendMessage(strconv.Itoa(userID), fmt.Sprintf("✅ درخواست شما برای گروه تأیید شد!\n🔗 لینک ورود: %s", link))
		api.AnswerCallback(query.ID, "✅ درخواست تأیید شد.", false)

		// به مالک هم اطلاع بده
		ownerID, _ := strconv.Atoi(config.OwnerID)
		api.SendMessage(strconv.Itoa(ownerID), fmt.Sprintf("📩 کاربر %d به گروه %d دعوت شد.", userID, chatIDInt))
	} else if strings.HasPrefix(data, "reject_invite:") {
		// رد دعوت
		api.SendMessage(strconv.Itoa(userID), "❌ متأسفانه درخواست شما برای ورود به گروه رد شد.")
		api.AnswerCallback(query.ID, "❌ درخواست رد شد.", false)
	}

	// حذف دکمه‌های پیام
	if query.Message != nil {
		api.EditMessageText(
			strconv.Itoa(query.Message.Chat.ID),
			query.Message.MessageID,
			query.Message.Text+"\n\n(پاسخ داده شد)",
			nil,
		)
	}
}

// تابع کمکی برای split کردن callback data
func splitCallbackData(data string) []string {
	result := []string{}
	current := ""
	for _, ch := range data {
		if ch == ':' {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
