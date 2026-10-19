package handlers

import (
	"fmt"
	"strconv"
	"time"

	"dalbot/api"
	"dalbot/models"
)

func handlePromote(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	if targetUserID == 0 {
		api.SendMessage(chatID, "❌ برای ترفیع، باید روی کاربر ریپلای کنید.")
		return
	}
	group, _ := models.GetGroup(chatIDInt)
	if group != nil && int64(targetUserID) == group.OwnerID {
		api.SendMessage(chatID, "❌ نمی‌توانید مالک گروه را ترفیع دهید.")
		return
	}
	level := models.LevelAdmin
	if len(args) > 0 && args[0] == "ارشد" {
		level = models.LevelSeniorAdmin
	}
	err := models.AddAdmin(chatIDInt, int64(targetUserID), level, int64(userID))
	if err != nil {
		api.SendMessage(chatID, "❌ خطا در ثبت ادمین.")
		return
	}
	title := "ادمین"
	if level == models.LevelSeniorAdmin {
		title = "ادمین ارشد"
	}
	msg, _ := api.SendMessage(chatID, fmt.Sprintf("👮 %s به مقام %s منصوب شد.",
		"["+targetFirstName+"](uid:"+strconv.Itoa(targetUserID)+")", title))
	if msg != nil {
		scheduleAutoDelete(chatIDInt, msg.MessageID, 10*time.Second)
	}
}

func handleDemote(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	if targetUserID == 0 {
		api.SendMessage(chatID, "❌ برای تنزیل، باید روی کاربر ریپلای کنید.")
		return
	}
	group, _ := models.GetGroup(chatIDInt)
	if group != nil && int64(targetUserID) == group.OwnerID {
		api.SendMessage(chatID, "❌ نمی‌توانید مالک گروه را تنزیل دهید.")
		return
	}
	if _, err := models.GetAdminLevel(chatIDInt, int64(targetUserID)); err != nil {
		api.SendMessage(chatID, "❌ این کاربر ادمین نیست.")
		return
	}
	if err := models.RemoveAdmin(chatIDInt, int64(targetUserID)); err != nil {
		api.SendMessage(chatID, "❌ خطا در حذف ادمین.")
		return
	}
	msg, _ := api.SendMessage(chatID, fmt.Sprintf("⬇️ %s از مقام مدیریت برکنار شد.",
		"["+targetFirstName+"](uid:"+strconv.Itoa(targetUserID)+")"))
	if msg != nil {
		scheduleAutoDelete(chatIDInt, msg.MessageID, 10*time.Second)
	}
}

func handleSpecial(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	if targetUserID == 0 {
		api.SendMessage(chatID, "❌ برای اعطای ویژه، باید روی کاربر ریپلای کنید.")
		return
	}
	group, _ := models.GetGroup(chatIDInt)
	if group != nil && int64(targetUserID) == group.OwnerID {
		api.SendMessage(chatID, "❌ مالک گروه نیازی به سطح ویژه ندارد.")
		return
	}
	err := models.AddAdmin(chatIDInt, int64(targetUserID), models.LevelSpecial, int64(userID))
	if err != nil {
		api.SendMessage(chatID, "❌ خطا در ثبت کاربر ویژه.")
		return
	}
	msg, _ := api.SendMessage(chatID, fmt.Sprintf("🌟 کاربر %s اکنون **ویژه** است و از تمام قفل‌ها معاف می‌باشد.",
		"["+targetFirstName+"](uid:"+strconv.Itoa(targetUserID)+")"))
	if msg != nil {
		scheduleAutoDelete(chatIDInt, msg.MessageID, 10*time.Second)
	}
}
