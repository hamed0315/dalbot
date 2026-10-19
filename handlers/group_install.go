package handlers

import (
	"fmt"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
	"dalbot/models"
)

func handleInstall(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	chatIDInt := parseInt64Chat(chatID)

	// 🛡️ احراز مالکیت واقعی (Creator)
	cm, err := api.GetChatMember(chatID, userID)
	if err != nil || cm == nil {
		sendGroupMsg(chatID, AutoDel5, "❌ برای نصب ربات، ابتدا باید ربات را **ادمین** کنید و دسترسی کامل بدهید.")
		return
	}
	if cm.Status != "creator" {
		sendGroupMsg(chatID, AutoDel5, "❌ فقط **مالک اصلی گروه (Creator)** می‌تواند ربات را نصب کند.\nشما در حال حاضر «"+cm.Status+"» هستید.")
		return
	}

	// 📛 دریافت نام واقعی گروه
	groupTitle := chatID
	if chatInfo, err := api.GetChat(chatID); err == nil && chatInfo.Title != "" {
		groupTitle = chatInfo.Title
	} else if err != nil {
		logger.Error.Printf("handleInstall: GetChat failed: %v", err)
	}

	// بررسی آیا گروه قبلاً نصب شده است؟
	group, _ := models.GetGroup(chatIDInt)
	if group != nil {
		// انتقال مالکیت (OwnerID عوض شده)
		if int64(userID) != group.OwnerID {
			err := models.SetGroupOwner(chatIDInt, int64(userID))
			if err != nil {
				sendGroupMsg(chatID, AutoDel5, "❌ خطا در به‌روزرسانی مالکیت. لطفاً دوباره تلاش کنید.")
				return
			}
			// انتقال مالکیت شمارنده را افزایش نمی‌دهد
			database.DB.Exec("INSERT OR REPLACE INTO approved_users (chat_id, user_id) VALUES (?, ?)", chatIDInt, userID)

			var username string
			row := database.DB.QueryRow("SELECT username FROM private_users WHERE user_id = ?", userID)
			if err := row.Scan(&username); err != nil || username == "" {
				username = ""
			}
			var ownerLine string
			if username != "" {
				ownerLine = fmt.Sprintf("👤 مالک جدید: @%s", username)
			} else {
				ownerLine = fmt.Sprintf("👤 مالک جدید: [%s](tg://user?id=%d)", adminFirstName, userID)
			}
			msg := fmt.Sprintf(
				"🔄 **بروزرسانی فرماندهی گروه**\n\n"+
					"🏰 گروه «%s» از این پس تحت فرماندهی شما قرار گرفت.\n\n"+
					"👑 **سرپرست جدید:** %s\n\n"+
					"✨ تمامی اختیارات مدیریتی دال برای شما فعال شد. برای دیدن قدرت کامل، `راهنما` را بفرستید.",
				groupTitle, ownerLine,
			)
			sendGroupMsg(chatID, AutoDel5, msg)
			return
		}

		// اگر همان مالک باشد، پیام تکراری
		sendGroupMsg(chatID, AutoDel5, "❌ این گروه قبلاً توسط شما نصب شده است.")
		return
	}

	// نصب جدید
	err = models.CreateGroup(chatIDInt, groupTitle, int64(userID))
	if err != nil {
		sendGroupMsg(chatID, AutoDel5, "❌ خطا در نصب ربات. لطفاً دوباره تلاش کنید.")
		return
	}
	database.DB.Exec("INSERT OR REPLACE INTO approved_users (chat_id, user_id) VALUES (?, ?)", chatIDInt, userID)

	group, _ = models.GetGroup(chatIDInt)
	groupTitle = chatID
	if group != nil && group.Title != "" {
		groupTitle = group.Title
	}

	var username string
	row := database.DB.QueryRow("SELECT username FROM private_users WHERE user_id = ?", userID)
	if err := row.Scan(&username); err != nil || username == "" {
		username = ""
	}
	var ownerLine string
	if username != "" {
		ownerLine = fmt.Sprintf("👤 مالک: @%s", username)
	} else {
		ownerLine = fmt.Sprintf("👤 مالک: [%s](tg://user?id=%d)", adminFirstName, userID)
	}

	msg := fmt.Sprintf(
		"🌟 **خوش آمدین به خانوادهٔ دال!**\n\n"+
			"🛡️ ربات دال هم‌اکنون در گروه «%s» فعال شد و نگهبانی نظم و امنیت این جمع رو بر عهده گرفت.\n\n"+
			"👑 **سرپرست محترم گروه:** %s\n\n"+
			"📌 برای مشاهدهٔ راهنمای کامل، یک `راهنما` بفرستید.",
		groupTitle, ownerLine,
	)
	sendGroupMsg(chatID, AutoDel5, msg)
}
