package handlers

import (
	"fmt"
	"strings"
	"time"

	"dalbot/api"
	"dalbot/database"
)

func handleStats(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	var sb strings.Builder

	sb.WriteString("📊 **۱۰ بن اخیر:**\n")
	banRows, _ := database.DB.Query("SELECT user_id, target_first_name, admin_id, admin_first_name, reason, created_at FROM ban_history WHERE chat_id = ? ORDER BY id DESC LIMIT 10", chatIDInt)
	if banRows != nil {
		defer banRows.Close()
		i := 0
		for banRows.Next() {
			var uid, aid int64
			var tName, aName, reason, createdAt string
			if err := banRows.Scan(&uid, &tName, &aid, &aName, &reason, &createdAt); err == nil {
				i++
				sb.WriteString(fmt.Sprintf("%d. [%s](uid:%d) توسط [%s](uid:%d) - %s (%s)\n", i, tName, uid, aName, aid, reason, createdAt))
			}
		}
		if i == 0 {
			sb.WriteString("_هیچ موردی یافت نشد._\n")
		}
	}
	sb.WriteString("\n")

	sb.WriteString("⚠️ **۱۰ اخطار اخیر:**\n")
	warnRows, _ := database.DB.Query("SELECT user_id, target_first_name, warned_by, by_first_name, reason, created_at FROM warnings WHERE chat_id = ? ORDER BY id DESC LIMIT 10", chatIDInt)
	if warnRows != nil {
		defer warnRows.Close()
		i := 0
		for warnRows.Next() {
			var uid, wid int64
			var tName, wName, reason, createdAt string
			if err := warnRows.Scan(&uid, &tName, &wid, &wName, &reason, &createdAt); err == nil {
				i++
				sb.WriteString(fmt.Sprintf("%d. [%s](uid:%d) توسط [%s](uid:%d) - %s (%s)\n", i, tName, uid, wName, wid, reason, createdAt))
			}
		}
		if i == 0 {
			sb.WriteString("_هیچ موردی یافت نشد._\n")
		}
	}
	sb.WriteString("\n")

	sb.WriteString("🔇 **۱۰ سکوت اخیر:**\n")
	muteRows, _ := database.DB.Query("SELECT user_id, target_first_name, admin_id, admin_first_name, duration_minutes, created_at FROM mute_history WHERE chat_id = ? ORDER BY id DESC LIMIT 10", chatIDInt)
	if muteRows != nil {
		defer muteRows.Close()
		i := 0
		for muteRows.Next() {
			var uid, aid int64
			var tName, aName string
			var duration int
			var createdAt string
			if err := muteRows.Scan(&uid, &tName, &aid, &aName, &duration, &createdAt); err == nil {
				i++
				sb.WriteString(fmt.Sprintf("%d. [%s](uid:%d) توسط [%s](uid:%d) - %d دقیقه (%s)\n", i, tName, uid, aName, aid, duration, createdAt))
			}
		}
		if i == 0 {
			sb.WriteString("_هیچ موردی یافت نشد._\n")
		}
	}
	sb.WriteString("\n")

	sb.WriteString("🏆 **فعال‌ترین ادمین‌ها (هفته اخیر):**\n")
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	activeRows, _ := database.DB.Query(`
		SELECT admin_id, admin_first_name, COUNT(*) as cnt FROM (
			SELECT admin_id, admin_first_name FROM ban_history WHERE chat_id = ? AND created_at >= ?
			UNION ALL
			SELECT warned_by, by_first_name FROM warnings WHERE chat_id = ? AND created_at >= ?
			UNION ALL
			SELECT admin_id, admin_first_name FROM mute_history WHERE chat_id = ? AND created_at >= ?
		) GROUP BY admin_id ORDER BY cnt DESC LIMIT 3`,
		chatIDInt, sevenDaysAgo, chatIDInt, sevenDaysAgo, chatIDInt, sevenDaysAgo)
	if activeRows != nil {
		defer activeRows.Close()
		i := 0
		for activeRows.Next() {
			var aid int64
			var aName string
			var cnt int
			if err := activeRows.Scan(&aid, &aName, &cnt); err == nil {
				i++
				sb.WriteString(fmt.Sprintf("%d. [%s](uid:%d) – %d اقدام\n", i, aName, aid, cnt))
			}
		}
		if i == 0 {
			sb.WriteString("_هیچ فعالیتی ثبت نشده._\n")
		}
	}

	api.SendMessage(chatID, sb.String())
}
