package handlers

import (
	"fmt"
	"strconv"
	"time"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
	"dalbot/models"
)

func handleCreateInviteLink(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	group, _ := models.GetGroup(chatIDInt)
	if group == nil {
		api.SendMessage(chatID, "❌ گروه ثبت نشده است.")
		return
	}
	if int64(userID) != group.OwnerID {
		level := getUserLevel(chatIDInt, userID)
		if level != "senior_admin" {
			api.SendMessage(chatID, "❌ فقط مالک یا ادمین ارشد می‌توانند لینک دعوت بسازند.")
			return
		}
	}

	// حذف خودکار کد قبلی گروه (اگر وجود دارد)
	database.DB.Exec("DELETE FROM invite_links WHERE chat_id = ?", chatIDInt)

	// ساخت کد جدید
	code := fmt.Sprintf("DAL-%04d", time.Now().UnixNano()%10000)
	_, err := database.DB.Exec("INSERT INTO invite_links (code, chat_id, owner_id) VALUES (?, ?, ?)", code, chatIDInt, userID)
	if err != nil {
		logger.Error.Printf("handleCreateInviteLink DB error: %v", err)
		api.SendMessage(chatID, "❌ خطا در ساخت لینک دعوت.")
		return
	}

	link := fmt.Sprintf("https://ble.ir/daal78bot?start=%s", code)
	msg := fmt.Sprintf("🔗 لینک دعوت هوشمند جدید شما:\n%s\n\n📌 این لینک را در لینکدونی‌ها قرار دهید.", link)
	api.SendMessage(strconv.Itoa(userID), msg)

	// پیام تأیید در گروه با زمان‌بندی حذف ۷ ثانیه
	confirmText := "✅ لینک دعوت هوشمند جدید با موفقیت ساخته شد و در پیام خصوصی برای شما ارسال گردید."
	sentMsg, err := api.SendMessage(chatID, confirmText)
	if err == nil && sentMsg != nil {
		scheduleAutoDelete(chatIDInt, sentMsg.MessageID, 7*time.Second)
	}
}
