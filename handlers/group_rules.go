package handlers

import (
	"strings"
	"time"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
)

func handleSetRules(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		api.SendMessage(chatID, "❌ برای ثبت قوانین، روی یک پیام متنی ریپلای کرده و دستور `تنظیم قوانین` را ارسال کنید.")
		return
	}
	ruleText := args[0]
	_, err := database.DB.Exec("INSERT OR REPLACE INTO group_rules (chat_id, rule_text) VALUES (?, ?)", chatIDInt, ruleText)
	if err != nil {
		logger.Error.Printf("handleSetRules DB error: %v", err)
		api.SendMessage(chatID, "❌ خطا در ثبت قوانین.")
		return
	}
	msg, _ := api.SendMessage(chatID, "✅ قوانین گروه با موفقیت ذخیره شد.")
	if msg != nil {
		scheduleAutoDelete(chatIDInt, msg.MessageID, 10*time.Second)
	}
}

func handleShowRules(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	row := database.DB.QueryRow("SELECT rule_text FROM group_rules WHERE chat_id = ?", chatIDInt)
	var ruleText string
	err := row.Scan(&ruleText)
	if err != nil {
		api.SendMessage(chatID, "⚠️ هنوز قوانینی برای این گروه ثبت نشده است.")
		return
	}
	api.SendMessage(chatID, "📜 **قوانین گروه:**\n\n"+ruleText)
}
