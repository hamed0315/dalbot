package handlers

import (
	"fmt"
	"strings"

	"dalbot/api"
	"dalbot/services"
)

func handleRanking(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)
	top := services.GetTopUsers(chatIDInt, 20)
	if len(top) == 0 {
		api.SendMessage(chatID, "📊 امروز هنوز رنکینگی ثبت نشده است.")
		return
	}

	// دریافت اطلاعات کاربران گروه برای نمایش نام
	allUsers := services.GetGroupUsers(chatIDInt)
	// ساخت یک مپ برای دسترسی سریع به اسم
	nameMap := make(map[int]string)
	for _, u := range allUsers {
		displayName := u.FirstName
		if displayName == "" {
			displayName = u.Username
		}
		if displayName == "" {
			displayName = fmt.Sprintf("کاربر %d", u.UserID)
		}
		nameMap[u.UserID] = displayName
	}

	var medals = []string{"🥇", "🥈", "🥉"}
	var sb strings.Builder
	for i, u := range top {
		userName, ok := nameMap[u.UserID]
		if !ok {
			userName = fmt.Sprintf("کاربر %d", u.UserID)
		}
		if i < 3 {
			sb.WriteString(fmt.Sprintf("%s رتبه %d: [%s](uid:%d) – %d پیام\n", medals[i], i+1, userName, u.UserID, u.Count))
		} else {
			sb.WriteString(fmt.Sprintf("%d. [%s](uid:%d) – %d پیام\n", i+1, userName, u.UserID, u.Count))
		}
	}
	api.SendMessage(chatID, "📊 **رنکینگ امروز:**\n\n"+sb.String())
}
