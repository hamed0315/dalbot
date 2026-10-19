package handlers

import (
	"fmt"
	"strconv"
	"time"

	"dalbot/api"
	"dalbot/database"
)

func handlePurge(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	if len(args) == 0 {
		sendGroupMsg(chatID, AutoDel5, "❌ لطفاً تعداد پیام‌ها را مشخص کنید. مثال: `پاکسازی ۱۰۰`")
		return
	}

	count, err := strconv.Atoi(args[0])
	if err != nil || count < 1 || count > 500 {
		sendGroupMsg(chatID, AutoDel5, "❌ تعداد باید یک عدد بین ۱ تا ۵۰۰ باشد.")
		return
	}

	chatIDInt := parseInt64Chat(chatID)

	// 📦 خواندن مستقیم از دیتابیس purge_queue (جدیدترین پیام‌ها)
	rows, err := database.DB.Query(
		"SELECT message_id FROM purge_queue WHERE chat_id = ? ORDER BY created_at DESC LIMIT ?",
		chatIDInt, count,
	)
	if err != nil {
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در خواندن لیست پیام‌ها.")
		return
	}
	defer rows.Close()

	var msgIDs []int
	for rows.Next() {
		var mid int
		if err := rows.Scan(&mid); err == nil {
			msgIDs = append(msgIDs, mid)
		}
	}

	if len(msgIDs) == 0 {
		sendGroupMsg(chatID, AutoDel5, "📭 پیامی برای پاکسازی یافت نشد.")
		return
	}

	// 🗑️ حذف پیام‌ها (Async)
	go func() {
		deleted := 0
		for _, msgID := range msgIDs {
			err := api.DeleteMessage(chatID, msgID)
			if err != nil {
				continue
			}
			// حذف از دیتابیس هم‌زمان
			database.DB.Exec("DELETE FROM purge_queue WHERE chat_id = ? AND message_id = ?", chatIDInt, msgID)
			deleted++
			time.Sleep(100 * time.Millisecond)
		}
		logOperation(chatIDInt, "پاکسازی", true, fmt.Sprintf("%d پیام حذف شد (درخواستی %d)", deleted, count))
	}()

	msg := fmt.Sprintf("🧹 عملیات پاکسازی برای %d پیام آغاز شد. لطفاً کمی صبر کنید...", len(msgIDs))
	sendGroupMsg(chatID, AutoDel5, msg)
}
