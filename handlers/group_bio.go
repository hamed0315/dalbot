package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"dalbot/api"
	"dalbot/database"
)

func handleSetBio(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)

	if targetUserID == 0 {
		api.SendMessage(chatID, "❌ برای ثبت اصل، باید روی پیام کاربر ریپلای کنید.")
		return
	}

	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		api.SendMessage(chatID, "❌ پیام مورد نظر متنی ندارد.")
		return
	}

	bioText := args[0]
	database.DB.Exec("INSERT OR REPLACE INTO user_bios (chat_id, user_id, bio_text, added_by) VALUES (?, ?, ?, ?)", chatIDInt, targetUserID, bioText, userID)

	msg, _ := api.SendMessage(chatID, fmt.Sprintf("✅ معرفی %s با موفقیت ثبت شد.",
		"["+targetFirstName+"](uid:"+strconv.Itoa(targetUserID)+")"))
	if msg != nil {
		scheduleAutoDelete(chatIDInt, msg.MessageID, 10*time.Second)
	}
}

func handleGetBio(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)

	if targetUserID == 0 {
		api.SendMessage(chatID, "❌ برای نمایش اصل، باید روی کاربر ریپلای کنید.")
		return
	}

	row := database.DB.QueryRow("SELECT bio_text FROM user_bios WHERE chat_id = ? AND user_id = ?", chatIDInt, targetUserID)
	var bioText string
	err := row.Scan(&bioText)
	if err != nil {
		api.SendMessage(chatID, "⚠️ هیچ معرفی‌نامه‌ای برای "+targetFirstName+" ثبت نشده است.")
		return
	}

	likeCount := getBioLikeCount(chatIDInt, int64(targetUserID))

	text := fmt.Sprintf("📝 معرفی %s:\n\n%s\n\n❤️ %d لایک",
		"["+targetFirstName+"](uid:"+strconv.Itoa(targetUserID)+")",
		bioText, likeCount)

	keyboard := api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{{Text: "💜 لایک", CallbackData: "like_bio:" + strconv.Itoa(targetUserID)}},
		},
	}
	api.SendMessageWithKeyboard(chatID, text, keyboard)
}
