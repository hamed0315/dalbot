package handlers

import (
	"fmt"
	"strings"
	"time"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
	"dalbot/services"
)

func sendForcedJoinMessage(chatID string, userID int, firstName string, missingChannels []services.ForcedChannel) {
	// متن پیام ساده و هدفمند
	text := fmt.Sprintf("🌸 %s عزیز!\nبرای ادامهٔ گفت‌وگو در این گروه، ابتدا از طریق دکمهٔ زیر در کانال عضو شوید.",
		firstName,
	)

	// فقط یک دکمه با عنوان «پورتال عضویت» برای اولین کانال (فعلاً یک کانال مجاز است)
	var buttons [][]api.InlineKeyboardButton
	if len(missingChannels) > 0 {
		ch := missingChannels[0]
		link := ch.Link
		if link == "" {
			link = "https://ble.ir/" + strings.TrimPrefix(ch.Username, "@")
		} else if !strings.HasPrefix(link, "http") {
			link = "https://" + link
		}
		buttons = append(buttons, []api.InlineKeyboardButton{
			{Text: "🔹 پورتال عضویت", URL: link},
		})
	}

	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	sentMsg, err := api.SendMessageWithKeyboard(chatID, text, keyboard)
	if err != nil {
		logger.Error.Printf("sendForcedJoinMessage failed: %v", err)
		return
	}
	chatIDInt := parseInt64Chat(chatID)
	scheduleAutoDelete(chatIDInt, sentMsg.MessageID, 4*time.Second)
}

func normalizeChannelUsername(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "@")
	return "@" + raw
}

func handleAddForcedChannel(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	if len(args) == 0 {
		conversationState[int64(userID)] = "awaiting_forced_channel_add"
		sendGroupMsg(chatID, AutoDel5, "📝 لطفاً نام کاربری کانال را بفرستید (مثلاً @test):")
		return
	}

	username := normalizeChannelUsername(args[0])
	logger.Info.Printf("Adding forced channel: %s", username)

	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM group_forced_channels WHERE chat_id = ?", chatIDInt).Scan(&count)
	if count >= 1 {
		sendGroupMsg(chatID, AutoDel5, "❌ این گروه در حال حاضر ۱ کانال اجباری دارد. برای افزودن کانال جدید، ابتدا کانال قبلی را حذف کنید.")
		return
	}

	if !services.IsBotMemberOfChannel(username) {
		sendGroupMsg(chatID, AutoDel5, "❌ ربات ابتدا باید در کانال ادمین شود.\nلطفاً ربات @Daal78bot را در کانال "+username+" ادمین کنید و سپس دوباره دستور را بفرستید.")
		return
	}

	chatInfo, err := api.GetChat(username)
	title := username
	if err == nil {
		title = chatInfo.Title
	}
	link, _ := services.GetJoinLinkForChannel(username)

	_, err = database.DB.Exec(
		"INSERT INTO group_forced_channels (chat_id, channel_username, channel_title, join_link, added_by) VALUES (?, ?, ?, ?, ?)",
		chatIDInt, username, title, link, userID,
	)
	if err != nil {
		logger.Error.Printf("handleAddForcedChannel DB error: %v", err)
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در ثبت کانال.")
		return
	}

	confirmText := fmt.Sprintf("✅ کانال %s به لیست عضویت اجباری اضافه شد.", title)
	sendGroupMsg(chatID, AutoDel60, confirmText)
}

func handleRemoveForcedChannel(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	if len(args) == 0 {
		conversationState[int64(userID)] = "awaiting_forced_channel_remove"
		sendGroupMsg(chatID, AutoDel5, "📝 لطفاً نام کاربری کانال را بفرستید (مثلاً @test):")
		return
	}

	username := normalizeChannelUsername(args[0])
	logger.Info.Printf("Removing forced channel: %s", username)

	result, err := database.DB.Exec("DELETE FROM group_forced_channels WHERE chat_id = ? AND channel_username = ?", chatIDInt, username)
	if err != nil {
		logger.Error.Printf("handleRemoveForcedChannel DB error: %v", err)
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در حذف کانال.")
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("⚠️ کانال %s در لیست عضویت اجباری یافت نشد.", username))
	} else {
		confirmText := fmt.Sprintf("✅ کانال %s از لیست عضویت اجباری حذف شد.", username)
		sendGroupMsg(chatID, AutoDel60, confirmText)
	}
}

func handleListForcedChannels(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	channels, err := services.GetForcedChannels(chatIDInt)
	if err != nil || len(channels) == 0 {
		sendGroupMsg(chatID, AutoDel5, "📊 هیچ کانال اجباری برای این گروه تعریف نشده است.")
		return
	}
	var sb strings.Builder
	sb.WriteString("📌 **کانال‌های اجباری گروه:**\n")
	for _, ch := range channels {
		sb.WriteString(fmt.Sprintf("• %s (%s)\n", ch.Title, ch.Username))
	}
	sendGroupMsg(chatID, AutoDel5, sb.String())
}
