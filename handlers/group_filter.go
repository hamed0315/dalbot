package handlers

import (
	"fmt"
	"strings"

	"dalbot/database"
)

func handleAddBadWord(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	if len(args) == 0 {
		sendGroupMsg(chatID, AutoDel5, "❌ لطفاً کلمهٔ مورد نظر را بنویسید.")
		return
	}
	word := strings.TrimSpace(args[0])
	chatIDInt := parseInt64Chat(chatID)
	_, err := database.DB.Exec("INSERT OR IGNORE INTO bad_words (chat_id, word) VALUES (?, ?)", chatIDInt, word)
	if err != nil {
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در افزودن فیلتر.")
		return
	}
	sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("✅ کلمهٔ '%s' به فیلتر اضافه شد.", word))
}

func handleRemoveBadWord(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	if len(args) == 0 {
		sendGroupMsg(chatID, AutoDel5, "❌ لطفاً کلمهٔ مورد نظر را بنویسید.")
		return
	}
	word := strings.TrimSpace(args[0])
	chatIDInt := parseInt64Chat(chatID)
	// حذف تمام کلماتی که پس از Trim شدن با این مقدار برابرند
	result, err := database.DB.Exec("DELETE FROM bad_words WHERE chat_id = ? AND TRIM(word) = ?", chatIDInt, word)
	if err != nil {
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در حذف فیلتر.")
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("⚠️ کلمهٔ '%s' در لیست فیلتر یافت نشد.", word))
		return
	}
	sendGroupMsg(chatID, AutoDel5, fmt.Sprintf("✅ کلمهٔ '%s' از فیلتر حذف شد.", word))
}

func handleListBadWords(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	rows, err := database.DB.Query("SELECT word FROM bad_words WHERE chat_id = ?", chatIDInt)
	if err != nil {
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در خواندن لیست فیلترها.")
		return
	}
	defer rows.Close()

	var words []string
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err == nil {
			words = append(words, w)
		}
	}
	if len(words) == 0 {
		sendGroupMsg(chatID, AutoDel5, "📭 هیچ کلمهٔ فیلترشده‌ای وجود ندارد.")
		return
	}

	msg := "📋 **کلمات فیلترشده:**\n"
	for _, w := range words {
		msg += fmt.Sprintf("• %s\n", w)
	}
	sendGroupMsg(chatID, AutoDel5, msg)
}
