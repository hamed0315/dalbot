package handlers

import (
	"fmt"

	"dalbot/api"
	"dalbot/database"
	"dalbot/models"
	"dalbot/services"
)

// ------------------------------------------------------------
//  توابع عمومی پنل (داشبورد جدید)
// ------------------------------------------------------------
func handlePanel(chatID string, userID, targetUserID int, targetFirstName, adminFirstName string, args []string, replyToID int, messageID int) {
	level := getUserLevel(parseInt64Chat(chatID), userID)
	if level != "super_owner" && level != "owner" && level != "senior_admin" {
		api.SendMessage(chatID, "⛔ فقط مالک و ادمین ارشد می‌توانند از پنل مدیریت استفاده کنند.")
		return
	}
	chatIDInt := parseInt64Chat(chatID)
	showMainPanel(chatID, 0, adminFirstName, chatIDInt, true)
}

func showMainPanel(chatID string, messageID int, adminFirstName string, chatIDInt int64, newMsg bool) {
	lockCount := countActiveLocks(chatIDInt)
	badWordCount := countBadWords(chatIDInt)
	cfg := services.GetAntispamConfig(chatIDInt)
	spamPunishment := punishmentToEmoji(cfg.Punishment)
	aiStatus := "غیرفعال"
	if cfg.AiEnabled {
		aiStatus = "فعال"
	}
	forcedChCount := countForcedChannels(chatIDInt)

	group, _ := models.GetGroup(chatIDInt)
	groupTitle := "گروه"
	if group != nil {
		groupTitle = group.Title
	}

	text := fmt.Sprintf(
		"⚙️ **پنل مدیریت دال**\n"+
			"👤 **ادمین:** %s\n"+
			"🏷 **گروه:** %s\n\n"+
			"📊 **خلاصهٔ وضعیت:**\n"+
			"• 🛡️ قفل‌های فعال: %d\n"+
			"• 📝 کلمات فیلتر: %d\n"+
			"• ⚡ جریمهٔ اسپم: %s\n"+
			"• 🧠 هوش مصنوعی: %s\n"+
			"• 🔗 جوین اجباری: %d",
		adminFirstName, groupTitle, lockCount, badWordCount,
		spamPunishment, aiStatus, forcedChCount,
	)

	keyboard := api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{
				{Text: "🛡️ قفل‌ها و جریمه‌ها", CallbackData: "panel:locks:main"},
			},
			{
				{Text: "👥 کاربران", CallbackData: "panel:users"},
				{Text: "🔗 جوین اجباری", CallbackData: "panel:forced_join"},
			},
			{
				{Text: "📊 وضعیت گروه", CallbackData: fmt.Sprintf("panel:status:%d", messageID)},
				{Text: "⚙️ تنظیمات گروه", CallbackData: "panel:settings"},
			},
			{
				{Text: "🧹 پاکسازی ۱۰۰ پیام", CallbackData: fmt.Sprintf("panel:quickpurge:%d", messageID)},
			},
			{
				{Text: "❓ راهنمای پنل", CallbackData: "panel:help"},
			},
			{
				{Text: "❌ بستن پنل", CallbackData: "panel:close"},
			},
		},
	}

	if newMsg {
		api.SendMessageWithKeyboard(chatID, text, keyboard)
	} else {
		api.EditMessageText(chatID, messageID, text, &keyboard)
	}
}

// ------------------------------------------------------------
//  لیست قفل‌ها (ایموجی وضعیت روی دکمه)
// ------------------------------------------------------------
func showLockList(chatID string, messageID int, chatIDInt int64) {
	locks := []struct {
		code string
		name string
	}{
		{"link", "لینک"}, {"forward", "فوروارد"}, {"ad", "تبلیغات"},
		{"photo", "عکس"}, {"video", "ویدئو"}, {"audio", "صوت"},
		{"longtext", "طولانی"}, {"sticker", "استیکر"}, {"gif", "گیف"},
		{"poll", "نظرسنجی"}, {"all", "گروه"},
	}

	text := "🛡️ **لیست قفل‌ها**\n\n🔴 = غیرفعال  |  🟢 = فعال\nبرای مدیریت هر قفل روی دکمهٔ آن کلیک کنید."

	var buttons [][]api.InlineKeyboardButton
	for i, lock := range locks {
		active := isLockActive(chatIDInt, lock.code)
		status := "🔴"
		if active {
			status = "🟢"
		}
		btnText := fmt.Sprintf("%s %s", status, lock.name)
		if i%2 == 0 {
			buttons = append(buttons, []api.InlineKeyboardButton{
				{Text: btnText, CallbackData: fmt.Sprintf("panel:lockstep:%s:actions", lock.code)},
			})
		} else {
			buttons[len(buttons)-1] = append(buttons[len(buttons)-1], api.InlineKeyboardButton{
				Text: btnText, CallbackData: fmt.Sprintf("panel:lockstep:%s:actions", lock.code)},
			)
		}
	}
	buttons = append(buttons, []api.InlineKeyboardButton{
		{Text: "❓ راهنمای قفل‌ها", CallbackData: "panel:locks:help"},
	})
	buttons = append(buttons, []api.InlineKeyboardButton{
		{Text: "🔙 بازگشت به منوی اصلی", CallbackData: "panel:main"},
	})

	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	api.EditMessageText(chatID, messageID, text, &keyboard)
}

// ------------------------------------------------------------
//  مدیریت یک قفل (نمایش + تغییر وضعیت + تنظیم جریمه)
// ------------------------------------------------------------
func showLockActionsNew(chatID string, messageID int, chatIDInt int64, lockCode string, userID int) {
	active := isLockActive(chatIDInt, lockCode)
	status := "🔴 غیرفعال"
	toggleText := "✅ فعال‌سازی"
	toggleCallback := fmt.Sprintf("panel:lockstep:%s:toggle_activate", lockCode)
	if active {
		status = "🟢 فعال"
		toggleText = "❌ غیرفعال‌سازی"
		toggleCallback = fmt.Sprintf("panel:lockstep:%s:toggle_deactivate", lockCode)
	}

	lp := services.GetLockPunishment(chatIDInt, lockCode)
	punishText := punishmentToEmoji(lp.Punishment)
	lockName := lockCodeToName(lockCode)
	desc := lockDescriptions[lockCode]

	text := fmt.Sprintf(
		"🛡️ **%s** (%s)\n\n"+
			"📝 %s\n\n"+
			"⚖️ جریمهٔ فعلی: %s\n"+
			"🔄 وضعیت: %s",
		lockName, lockCode, desc, punishText, status,
	)

	buttons := [][]api.InlineKeyboardButton{
		{{Text: toggleText, CallbackData: toggleCallback}},
	}
	if active {
		buttons = append(buttons, []api.InlineKeyboardButton{
			{Text: "⚖️ تنظیم جریمه", CallbackData: fmt.Sprintf("panel:lockstep:%s:punish_select", lockCode)},
		})
	}
	buttons = append(buttons, []api.InlineKeyboardButton{
		{Text: "❓ توضیح این قفل", CallbackData: fmt.Sprintf("panel:lockstep:%s:info", lockCode)},
	})
	buttons = append(buttons, []api.InlineKeyboardButton{
		{Text: "🔙 بازگشت به لیست", CallbackData: "panel:locks:main"},
	})

	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	api.EditMessageText(chatID, messageID, text, &keyboard)
}

// ============================================================
//  توابع کمکی آمار
// ============================================================
func countActiveLocks(chatID int64) int {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM locks WHERE chat_id = ? AND is_locked = 1", chatID).Scan(&count)
	return count
}

func countBadWords(chatID int64) int {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM bad_words WHERE chat_id = ?", chatID).Scan(&count)
	return count
}

func countForcedChannels(chatID int64) int {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM group_forced_channels WHERE chat_id = ?", chatID).Scan(&count)
	return count
}

// ============================================================
//  جداول و توابع مشترک
// ============================================================
var lockDescriptions = map[string]string{
	"link":     "هرگونه لینک (http, ble.ir, t.me) و منشن‌ها (مانند @username یا @channel) را حذف می‌کند.\n📌 مثال: https://example.com یا @nobino78",
	"forward":  "پیام‌های فورواردی را حذف می‌کند.",
	"ad":       "کلمات کلیدی تبلیغاتی مانند «کسب درآمد», «فالوور», «ربات رایگان» و ... را شناسایی و حذف می‌کند.\n📌 این لیست به‌روزرسانی می‌شود.",
	"photo":    "تصاویر را حذف می‌کند.",
	"video":    "ویدئوها را حذف می‌کند.",
	"audio":    "فایل‌های صوتی و پیام‌های صوتی را حذف می‌کند.",
	"longtext": "متن‌های بیشتر از ۵۰۰ کاراکتر را حذف می‌کند.",
	"sticker":  "استیکرها را حذف می‌کند.",
	"gif":      "تصاویر متحرک (گیف) را حذف می‌کند.",
	"poll":     "نظرسنجی‌ها را حذف می‌کند (حتی ناشناس).",
	"all":      "تمام پیام‌های کاربران عادی را حذف می‌کند.",
}

func isLockActive(chatID int64, lockCode string) bool {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM locks WHERE chat_id = ? AND lock_type = ? AND is_locked = 1", chatID, lockCode).Scan(&count)
	return count > 0
}

func lockCodeToName(code string) string {
	names := map[string]string{
		"link": "لینک", "forward": "فوروارد", "ad": "تبلیغات",
		"photo": "عکس", "video": "ویدئو", "audio": "صوت",
		"longtext": "طولانی", "sticker": "استیکر", "gif": "گیف",
		"poll": "نظرسنجی", "all": "گروه",
	}
	if n, ok := names[code]; ok {
		return n
	}
	return code
}

func punishmentToEmoji(p string) string {
	switch p {
	case "delete": return "🗑️ فقط حذف"
	case "warn": return "⚠️ اخطار"
	case "mute": return "🔇 سکوت"
	case "ban": return "⛔ بن"
	default: return "🗑️ فقط حذف"
	}
}

// ------------------------------------------------------------
//  بخش‌های ساده‌تر پنل (کاربران، جوین اجباری، وضعیت، تنظیمات)
// ------------------------------------------------------------
func showUsersPanel(chatID string, messageID int) {
	text := "👥 **مدیریت کاربران**\n\n" +
		"برای بن، سکوت یا اخطار، روی پیام کاربر ریپلای بزنید و دستورات زیر را تایپ کنید:\n\n" +
		"`.بن`  `.سکوت ۶۰`  `.اخطار`  `.انبن`\n\n" +
		"برای پاک کردن اخطارها از دکمه زیر استفاده کنید:"
	buttons := [][]api.InlineKeyboardButton{
		{{Text: "🧹 حذف اخطارهای کاربر", CallbackData: fmt.Sprintf("panel:users:clearwarns:%d", messageID)}},
		{{Text: "🔙 بازگشت", CallbackData: "panel:main"}},
	}
	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	api.EditMessageText(chatID, messageID, text, &keyboard)
}

func showForcedJoinPanel(chatID string, messageID int, chatIDInt int64) {
	channels, _ := services.GetForcedChannels(chatIDInt)
	text := "🔗 **جوین اجباری**\n\n"
	if len(channels) > 0 {
		for _, ch := range channels {
			text += fmt.Sprintf("• %s (%s)\n", ch.Title, ch.Username)
		}
	} else {
		text += "هیچ کانال اجباری تعریف نشده است.\n"
	}
	buttons := [][]api.InlineKeyboardButton{
		{
			{Text: "➕ افزودن کانال", CallbackData: fmt.Sprintf("panel:forced_join:add:%d", messageID)},
			{Text: "🗑️ حذف کانال", CallbackData: fmt.Sprintf("panel:forced_join:remove:%d", messageID)},
		},
		{
			{Text: "🔙 بازگشت", CallbackData: "panel:main"},
		},
	}
	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	api.EditMessageText(chatID, messageID, text, &keyboard)
}

func showGroupStatusPanel(chatID string, messageID int, chatIDInt int64) {
	group, err := models.GetGroup(chatIDInt)
	if err != nil || group == nil {
		api.EditMessageText(chatID, messageID, "❌ اطلاعات گروه یافت نشد.", nil)
		return
	}
	ownerID := group.OwnerID

	text := "📊 **وضعیت گروه**\n\n"
	if services.IsUserInChannels(int(ownerID)) {
		text += "✅ مالک گروه در کانال‌های حمایتی **عضو** است.\n"
		text += "🔹 @nobino78\n🔹 @linkdoni78\n🔹 @xau_78\n\n"
		text += "🟢 ربات **فعال** است."
	} else {
		text += "⚠️ مالک گروه در کانال‌های حمایتی **عضو نیست**.\n"
		text += "🔹 @nobino78\n🔹 @linkdoni78\n🔹 @xau_78\n\n"
		text += "🔴 ربات **غیرفعال** شده است."
	}
	buttons := [][]api.InlineKeyboardButton{
		{
			{Text: "🔹 عضویت در کانال ۱", URL: "https://ble.ir/nobino78"},
			{Text: "🔹 عضویت در کانال ۲", URL: "https://ble.ir/linkdoni78"},
		},
		{
			{Text: "🔹 عضویت در کانال ۳", URL: "https://ble.ir/xau_78"},
		},
		{
			{Text: "🔙 بازگشت", CallbackData: "panel:main"},
		},
	}
	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	api.EditMessageText(chatID, messageID, text, &keyboard)
}

func showGroupSettingsPanel(chatID string, messageID int) {
	text := "⚙️ **تنظیمات گروه**\n\nانتخاب کنید:"
	buttons := [][]api.InlineKeyboardButton{
		{
			{Text: "📜 تنظیم قوانین", CallbackData: fmt.Sprintf("panel:settings:rules:%d", messageID)},
			{Text: "🌸 تنظیم خوش‌آمدگویی", CallbackData: fmt.Sprintf("panel:settings:welcome:%d", messageID)},
		},
		{
			{Text: "🔙 بازگشت", CallbackData: "panel:main"},
		},
	}
	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	api.EditMessageText(chatID, messageID, text, &keyboard)
}
