package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"dalbot/api"
	"dalbot/config"
	"dalbot/database"
	"dalbot/logger"
	"dalbot/models"
	"dalbot/services"
)

// ============================================================
//  چت خصوصی
// ============================================================
func processPrivate(msg *api.Message) {
	userID := msg.From.ID
	chatIDStr := strconv.Itoa(msg.Chat.ID)
	text := strings.TrimSpace(msg.Text)

	savePrivateUser(msg.From)

	// 🔐 دستور مخفی آموزش اسپم (فقط برای super_owner)
	if strconv.Itoa(userID) == config.OwnerID && text == "ثبت اسپم" {
		logger.Info.Printf("SuperOwner (%d) is teaching SPAM", userID)
		if msg.ReplyTo != nil {
			spamText := msg.ReplyTo.Text
			if spamText == "" {
				spamText = msg.ReplyTo.Caption
			}
			if spamText != "" {
				_, err := database.DB.Exec("INSERT INTO spam_corpus (chat_id, text, label) VALUES (0, ?, 'spam')", spamText)
				if err != nil {
					api.SendMessage(chatIDStr, "❌ خطا در ذخیره‌سازی: "+err.Error())
					logger.Error.Printf("SPAM DB Error: %v", err)
				} else {
					api.SendMessage(chatIDStr, "✅ دادهٔ اسپم با موفقیت به مدل یادگیری اضافه شد.")
					logger.Info.Println("SPAM Data stored successfully")
				}
			} else {
				api.SendMessage(chatIDStr, "⚠️ پیام ریپلای‌شده متنی ندارد.")
			}
		} else {
			api.SendMessage(chatIDStr, "⚠️ لطفاً روی یک پیام ریپلای بزنید و سپس «ثبت اسپم» را ارسال کنید.")
		}
		return
	}

	// پنل مالک
	if strconv.Itoa(userID) == config.OwnerID && text == "پنل" {
		showOwnerPanel(chatIDStr)
		return
	}

	// ادامهٔ مکالمات چندمرحله‌ای
	if strconv.Itoa(userID) == config.OwnerID {
		state, exists := conversationState[int64(userID)]
		if exists {
			handleConversationState(int64(userID), state, text)
			return
		}
	}

	// لینک دعوت عمیق
	if strings.HasPrefix(text, "/start ") {
		parts := strings.Fields(text)
		if len(parts) == 2 && strings.HasPrefix(parts[1], "DAL-") {
			code := parts[1]
			handleInviteDeepLink(chatIDStr, userID, code, msg.From.FirstName, msg.From.Username)
			return
		}
	}

	// خوش‌آمدگویی
	if text == "/start" || text == "start" {
		sendWelcomeMessage(chatIDStr, msg.From.FirstName)
	}
}

func sendWelcomeMessage(chatID string, firstName string) {
	api.SendSticker(chatID, "CAACAgIAAxkBAAEKmBdmD1QrAAHkZ1QxWJ8HQkhIx0pICwACLwADUOMjFk4vGqVvZcM8HgQ")
	message := fmt.Sprintf(
		"🌟 %s عزیز، به **داڵ، محافظ هوشمند گروه‌ها** خوش آمدی!\n\n"+
			"من قدرتمندترین دستیار مدیریت گروه در بله هستم و با مجموعه‌ای از ابزارهای پیشرفته، آرامش و امنیت را برای گروهت به ارمغان می‌آورم. 🛡️\n\n"+
			"✨ **توانایی‌های کلیدی من:**\n"+
			"🔹 **قفل نظرسنجی (حتی ناشناس!)** – فقط دال از پسش برمیاد\n"+
			"🔹 ۹ نوع قفل محتوا + قفل گروه\n"+
			"🔹 ضداسپم هوشمند با جریمهٔ قابل تنظیم\n"+
			"🔹 فیلتر هوشمند فوق‌سریع (بدون مصرف توکن)\n"+
			"🔹 سیستم اخطار و بن خودکار (بن، کیک، صیک)\n"+
			"🔹 سکوت و رفع سکوت (دستی و خودکار)\n"+
			"🔹 انبن (آزاد کردن کاربران بن‌شده با یوزرنیم یا ID)\n"+
			"🔹 حذف اخطارهای کاربر\n"+
			"🔹 سلسله‌مراتب مدیریت (مالک، ادمین ارشد، ادمین، ویژه)\n"+
			"🔹 ثبت و نمایش معرفی کاربران + لایک 💜\n"+
			"🔹 عضویت اجباری در کانال دلخواه (۱ کانال)\n"+
			"🔹 پاک‌سازی سریع پیام‌ها (تا ۵۰۰ پیام)\n"+
			"🔹 رتبه‌بندی روزانه کاربران 🏆\n"+
			"🔹 آمار پیشرفته (۱۰ بن، اخطار، سکوت اخیر + فعال‌ترین ادمین‌ها)\n"+
			"🔹 قوانین گروهی، فیلتر کلمات ممنوع\n"+
			"🔹 پین و حذف پین پیام\n"+
			"🔹 تگ همه و تگ فعالان\n"+
			"🔹 لینک دعوت هوشمند با تأیید مالک\n"+
			"🔹 پاک‌سازی خودکار پیام‌های من (گروه همیشه تمیز)\n"+
			"🔹 و ده‌ها قابلیت حرفه‌ای دیگر...\n\n"+
			"📌 **جهت استفاده رایگان و بدون محدودیت، در کانال‌های اسپانسر عضو شوید:** 👇",
		firstName,
	)
	keyboard := api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{{Text: "🔹 عضویت در کانال ۱", URL: "https://ble.ir/nobino78"}},
			{{Text: "🔹 عضویت در کانال ۲", URL: "https://ble.ir/linkdoni78"}},
			{{Text: "🔹 عضویت در کانال ۳", URL: "https://ble.ir/xau_78"}},
			{{Text: "✅ بررسی عضویت", CallbackData: "check_membership"}},
		},
	}
	api.SendMessageWithKeyboard(chatID, message, keyboard)
}

// ============================================================
//  مدیریت callbackهای عمومی (غیر پنلی)
// ============================================================
func handleCallback(query *api.CallbackQuery) {
	data := query.Data

	if strings.HasPrefix(data, "approve_invite:") || strings.HasPrefix(data, "reject_invite:") {
		handleApprovalCallback(query)
		return
	}

	chatID := strconv.Itoa(query.From.ID)

	if strings.HasPrefix(data, "tag:") {
		chatID := strconv.Itoa(query.Message.Chat.ID)
		parts := strings.Split(data, ":")
		if len(parts) == 2 {
			switch parts[1] {
			case "all":
				performTagAll(chatID)
			case "active":
				performTagActive(chatID, 24)
			}
		}
		api.AnswerCallback(query.ID, "", false)
		return
	}

	if strings.HasPrefix(data, "invite_menu:") {
		chatID := strconv.Itoa(query.Message.Chat.ID)
		parts := strings.Split(data, ":")
		if len(parts) == 2 {
			switch parts[1] {
			case "smart":
				handleCreateInviteLink(chatID, query.From.ID, 0, "", query.From.FirstName, nil, 0, query.Message.MessageID)
			case "show_smart":
				handleShowSmartLink(chatID, query.From.ID)
			case "normal":
				handleLink(chatID, query.From.ID, 0, "", query.From.FirstName, nil, 0, query.Message.MessageID)
			}
		}
		api.AnswerCallback(query.ID, "", false)
		return
	}

	if strings.HasPrefix(data, "help:") {
		chatID := strconv.Itoa(query.Message.Chat.ID)
		parts := strings.Split(data, ":")
		if len(parts) >= 2 {
			section := parts[1]
			if section == "main" && len(parts) == 3 {
				msgID, _ := strconv.Atoi(parts[2])
				showHelpMenu(chatID, msgID, query.From.FirstName)
			} else if section == "close" {
				api.DeleteMessage(chatID, query.Message.MessageID)
				api.AnswerCallback(query.ID, "", false)
				return
			} else {
				showHelpSection(chatID, query.Message.MessageID, section, query.From.FirstName)
			}
		}
		api.AnswerCallback(query.ID, "", false)
		return
	}

	if data == "panel_broadcast_groups" {
		conversationState[int64(query.From.ID)] = "awaiting_broadcast_group"
		api.SendMessage(chatID, "📝 لطفاً متن پیام را برای ارسال به **همهٔ گروه‌ها** بنویسید:")
		api.AnswerCallback(query.ID, "", false)
		return
	}
	if data == "panel_broadcast_users" {
		conversationState[int64(query.From.ID)] = "awaiting_broadcast_user"
		api.SendMessage(chatID, "📝 لطفاً متن پیام را برای ارسال به **همهٔ کاربران** بنویسید:")
		api.AnswerCallback(query.ID, "", false)
		return
	}
	if data == "panel_stats" {
		gCount, _ := getActiveGroupCount()
		uCount := getPrivateUserCount()
		text := fmt.Sprintf("📊 آمار ربات دال:\n\n👥 تعداد گروه‌های فعال: %d\n👤 کاربران خصوصی: %d", gCount, uCount)
		api.SendMessage(chatID, text)
		api.AnswerCallback(query.ID, "", false)
		return
	}

	if strings.HasPrefix(data, "like_bio:") {
		targetUserIDStr := strings.TrimPrefix(data, "like_bio:")
		targetUserID, _ := strconv.Atoi(targetUserIDStr)
		if query.Message != nil && query.Message.Chat.ID != 0 {
			chatIDStr := strconv.Itoa(query.Message.Chat.ID)
			likeBio(chatIDStr, query.From.ID, targetUserID, query.Message.MessageID)
		}
		api.AnswerCallback(query.ID, "👍 لایک شما ثبت شد.", false)
		return
	}

	if data == "check_membership" {
		if services.IsUserInChannels(query.From.ID) {
			api.SendSticker(chatID, "CAACAgIAAxkBAAEKmBlmD1RhAAHkZ3gxWJ8HQkhIx0pICwACMAADUOMjFk4vGqZvZcM8HgQ")
			text := fmt.Sprintf(
				"🎉 تبریک! تو عضو کانال‌های ما هستی و اکنون می‌تونی از ربات دال **رایگان و بدون محدودیت** استفاده کنی. 🚀\n\n"+
					"📌 **راهنمای نصب سریع:**\n"+
					"1️⃣ ربات را به گروه خود اضافه کن.\n"+
					"2️⃣ در گروه، کلمهٔ **نصب** را بفرست.\n"+
					"3️⃣ دستور **راهنما** را بزن.\n\n"+
					"✨ قدرت گرفته توسط تیم ۷۸ : @deu_78",
			)
			activeGroups, _ := getActiveGroupCount()
			companionsText := fmt.Sprintf("🔮 همراهان ما (%d گروه)", activeGroups)
			keyboard := api.InlineKeyboardMarkup{
				InlineKeyboard: [][]api.InlineKeyboardButton{
					{{Text: "💬 پشتیبانی", URL: "https://ble.ir/support78"}},
					{{Text: "📢 کانال رسمی", URL: "https://ble.ir/dal_bot"}},
					{{Text: companionsText, CallbackData: "show_companions"}},
				},
			}
			api.SendMessageWithKeyboard(chatID, text, keyboard)
		} else {
			text := "⚠️ عضویت شما کامل نیست.\n\nلطفاً در کانال‌های زیر عضو شوید:\n@nobino78\n@linkdoni78\n@xau_78"
			keyboard := api.InlineKeyboardMarkup{
				InlineKeyboard: [][]api.InlineKeyboardButton{
					{{Text: "🔹 کانال ۱", URL: "https://ble.ir/nobino78"}},
					{{Text: "🔹 کانال ۲", URL: "https://ble.ir/linkdoni78"}},
					{{Text: "🔹 کانال ۳", URL: "https://ble.ir/xau_78"}},
					{{Text: "✅ بررسی مجدد", CallbackData: "check_membership"}},
				},
			}
			api.SendMessageWithKeyboard(chatID, text, keyboard)
		}
		api.AnswerCallback(query.ID, "", false)
		return
	}

	if data == "restart" {
		sendWelcomeMessage(chatID, query.From.FirstName)
		api.AnswerCallback(query.ID, "", false)
		return
	}

	if strings.HasPrefix(data, "check_forced_join:") {
		chatIDForced := strings.TrimPrefix(data, "check_forced_join:")
		chatIDInt, _ := strconv.ParseInt(chatIDForced, 10, 64)
		userID := query.From.ID
		channels, _ := services.GetForcedChannels(chatIDInt)
		allJoined := true
		for _, ch := range channels {
			if !services.IsUserInChannel(ch.Username, userID) {
				allJoined = false
				break
			}
		}
		if allJoined {
			api.AnswerCallback(query.ID, "✅ عضویت تأیید شد.", true)
			if query.Message != nil {
				api.DeleteMessage(chatIDForced, query.Message.MessageID)
			}
		} else {
			api.AnswerCallback(query.ID, "⚠️ هنوز عضو نشده‌اید.", true)
		}
		return
	}

	if strings.HasPrefix(data, "panel:") {
		handlePanelCallback(query)
		return
	}
}

// ------------------------------------------------------------
//  درخواست ورود از طریق لینک هوشمند
// ------------------------------------------------------------
func handleInviteDeepLink(chatID string, userID int, code string, firstName string, username string) {
	var chatIDInt int64
	var creatorID int64
	err := database.DB.QueryRow("SELECT chat_id, owner_id FROM invite_links WHERE code = ?", code).Scan(&chatIDInt, &creatorID)
	if err != nil {
		api.SendMessage(chatID, "❌ لینک نامعتبر است.")
		return
	}

	if isUserGloballyBanned(int64(userID)) {
		api.SendMessage(chatID, "⛔ شما امکان ورود ندارید.")
		return
	}

	group, err := models.GetGroup(chatIDInt)
	if err != nil || group == nil {
		api.SendMessage(chatID, "❌ گروه یافت نشد.")
		return
	}
	groupTitle := group.Title
	ownerID := group.OwnerID

	userLine := fmt.Sprintf("👤 **کاربر:** %s\n", formatMention(userID, username, firstName))
	ownerMsg := fmt.Sprintf(
		"📩 **درخواست ورود**\n\n"+
			userLine+
			"🏷 **گروه:** %s\n\n"+
			"برای تأیید یا رد از دکمه‌ها استفاده کنید.",
		groupTitle,
	)
	approveData := fmt.Sprintf("approve_invite:%d:%d", chatIDInt, userID)
	rejectData := fmt.Sprintf("reject_invite:%d:%d", chatIDInt, userID)

	keyboard := api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{{Text: "✅ تأیید", CallbackData: approveData}, {Text: "❌ رد", CallbackData: rejectData}},
		},
	}

	if ownerID != 0 {
		api.SendMessageWithKeyboard(strconv.FormatInt(ownerID, 10), ownerMsg, keyboard)
	}
	adminRows, _ := database.DB.Query("SELECT user_id FROM group_admins WHERE chat_id = ? AND level = 'senior_admin'", chatIDInt)
	if adminRows != nil {
		defer adminRows.Close()
		for adminRows.Next() {
			var adminID int64
			if err := adminRows.Scan(&adminID); err == nil && adminID != ownerID {
				api.SendMessageWithKeyboard(strconv.FormatInt(adminID, 10), ownerMsg, keyboard)
			}
		}
	}

	api.SendMessage(chatID, "✅ درخواست شما ثبت شد. پس از تأیید، لینک گروه برایتان ارسال می‌شود.")
}
