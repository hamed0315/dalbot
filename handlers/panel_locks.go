package handlers

import (
	"fmt"

	"dalbot/api"
	"dalbot/database"
	"dalbot/services"
)

// ============================================================
//  LockSession / SpamSession
// ============================================================
type LockSession struct {
	ChatID    int64
	LockCode  string
	Step      string
	NewPunish string
	IsActive  bool
}

type SpamSession struct {
	ChatID    int64
	Step      string
	NewPunish string
}

var (
	lockSessions = make(map[int64]map[int64]LockSession)
	spamSessions = make(map[int64]map[int64]SpamSession)
)

func saveLockSession(userID, chatID int64, s LockSession) {
	if lockSessions[userID] == nil {
		lockSessions[userID] = make(map[int64]LockSession)
	}
	lockSessions[userID][chatID] = s
}

func getLockSession(userID, chatID int64) (LockSession, bool) {
	if m, ok := lockSessions[userID]; ok {
		s, ok := m[chatID]
		return s, ok
	}
	return LockSession{}, false
}

func deleteLockSession(userID, chatID int64) {
	if m, ok := lockSessions[userID]; ok {
		delete(m, chatID)
	}
}

func saveSpamSession(userID, chatID int64, s SpamSession) {
	if spamSessions[userID] == nil {
		spamSessions[userID] = make(map[int64]SpamSession)
	}
	spamSessions[userID][chatID] = s
}

func getSpamSession(userID, chatID int64) (SpamSession, bool) {
	if m, ok := spamSessions[userID]; ok {
		s, ok := m[chatID]
		return s, ok
	}
	return SpamSession{}, false
}

func deleteSpamSession(userID, chatID int64) {
	if m, ok := spamSessions[userID]; ok {
		delete(m, chatID)
	}
}

// ============================================================
//  بخش ضداسپم (پنل)
// ============================================================




func showAiConfirm(chatID string, messageID int, chatIDInt int64) {
	cfg := services.GetAntispamConfig(chatIDInt)
	actionText := "✅ فعال‌سازی"
	if cfg.AiEnabled {
		actionText = "❌ غیرفعال‌سازی"
	}
	text := fmt.Sprintf("🧠 **هوش مصنوعی**\n\nوضعیت فعلی: %s\n\nآیا می‌خواهید وضعیت را تغییر دهید؟", map[bool]string{true: "فعال", false: "غیرفعال"}[cfg.AiEnabled])
	buttons := [][]api.InlineKeyboardButton{
		{
			{Text: actionText, CallbackData: fmt.Sprintf("panel:antispam:ai_toggle:%d", messageID)},
			{Text: "🔙 بازگشت", CallbackData: fmt.Sprintf("panel:antispam:%d", messageID)},
		},
	}
	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	api.EditMessageText(chatID, messageID, text, &keyboard)
}

// ============================================================
//  State Machine قفل‌ها (lockstep)
// ============================================================
func handleLockStep(chatID string, messageID int, chatIDInt int64, query *api.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		return
	}
	lockCode := parts[2]
	step := ""
	if len(parts) > 3 {
		step = parts[3]
	}
	userID := query.From.ID
	msgID := query.Message.MessageID

	switch step {
	case "actions":
		showLockActionsNew(chatID, msgID, chatIDInt, lockCode, userID)
		api.AnswerCallback(query.ID, "", false)

	case "toggle_activate":
		saveLockSession(int64(userID), chatIDInt, LockSession{
			ChatID:    chatIDInt,
			LockCode:  lockCode,
			Step:      "punish_select",
			NewPunish: "",
			IsActive:  false,
		})
		api.AnswerCallback(query.ID, "جریمه مورد نظر را انتخاب کنید", false)
		showLockPunishSelect(chatID, msgID, chatIDInt, lockCode, userID)

	case "toggle_deactivate":
		database.DB.Exec("DELETE FROM locks WHERE chat_id = ? AND lock_type = ?", chatIDInt, lockCode)
		services.InvalidateLocksCache(chatIDInt)
		api.AnswerCallback(query.ID, "❌ قفل غیرفعال شد", false)
		showLockActionsNew(chatID, msgID, chatIDInt, lockCode, userID)

	case "punish_select":
		active := isLockActive(chatIDInt, lockCode)
		saveLockSession(int64(userID), chatIDInt, LockSession{
			ChatID:    chatIDInt,
			LockCode:  lockCode,
			Step:      "punish_select",
			NewPunish: "",
			IsActive:  active,
		})
		showLockPunishSelect(chatID, msgID, chatIDInt, lockCode, userID)
		api.AnswerCallback(query.ID, "", false)

	case "confirm":
		if len(parts) < 5 {
			return
		}
		punishment := parts[4]
		sess, ok := getLockSession(int64(userID), chatIDInt)
		if !ok {
			api.AnswerCallback(query.ID, "⏱ جلسه منقضی شده، دوباره تلاش کنید.", true)
			showLockActionsNew(chatID, msgID, chatIDInt, lockCode, userID)
			return
		}
		sess.Step = "confirm"
		sess.NewPunish = punishment
		saveLockSession(int64(userID), chatIDInt, sess)
		showLockConfirm(chatID, msgID, chatIDInt, lockCode, userID, sess)
		api.AnswerCallback(query.ID, "", false)

	case "apply":
		sess, ok := getLockSession(int64(userID), chatIDInt)
		if !ok {
			api.AnswerCallback(query.ID, "⏱ جلسه منقضی شده، دوباره تلاش کنید.", true)
			showLockActionsNew(chatID, msgID, chatIDInt, lockCode, userID)
			return
		}
		if !sess.IsActive {
			database.DB.Exec("INSERT OR REPLACE INTO locks (chat_id, lock_type, is_locked) VALUES (?, ?, 1)", chatIDInt, lockCode)
			services.InvalidateLocksCache(chatIDInt)
		}
		services.SetLockPunishment(chatIDInt, lockCode, sess.NewPunish, 1)
		msg := fmt.Sprintf("✅ قفل %s با جریمه %s فعال شد", lockCodeToName(lockCode), punishmentToEmoji(sess.NewPunish))
		if sess.IsActive {
			msg = fmt.Sprintf("✅ جریمه %s برای قفل %s تنظیم شد", punishmentToEmoji(sess.NewPunish), lockCodeToName(lockCode))
		}
		api.AnswerCallback(query.ID, msg, true)
		deleteLockSession(int64(userID), chatIDInt)
		showLockActionsNew(chatID, msgID, chatIDInt, lockCode, userID)

	case "info":
		desc := lockDescriptions[lockCode]
		api.AnswerCallback(query.ID, desc, true)

	default:
		showLockActionsNew(chatID, msgID, chatIDInt, lockCode, userID)
		api.AnswerCallback(query.ID, "", false)
	}
}

func showLockPunishSelect(chatID string, messageID int, chatIDInt int64, lockCode string, userID int) {
	lp := services.GetLockPunishment(chatIDInt, lockCode)
	lockName := lockCodeToName(lockCode)
	text := fmt.Sprintf("⚖️ **جریمهٔ قفل %s**\n\nفعلی: %s\n\nجریمهٔ جدید را انتخاب کنید:", lockName, punishmentToEmoji(lp.Punishment))
	buttons := [][]api.InlineKeyboardButton{
		{{Text: "🗑️ فقط حذف", CallbackData: fmt.Sprintf("panel:lockstep:%s:confirm:delete", lockCode)}},
		{{Text: "⚠️ اخطار", CallbackData: fmt.Sprintf("panel:lockstep:%s:confirm:warn", lockCode)}},
		{{Text: "🔇 سکوت (۱ ساعت)", CallbackData: fmt.Sprintf("panel:lockstep:%s:confirm:mute", lockCode)}},
		{{Text: "⛔ بن", CallbackData: fmt.Sprintf("panel:lockstep:%s:confirm:ban", lockCode)}},
		{{Text: "🔙 بازگشت", CallbackData: fmt.Sprintf("panel:lockstep:%s:actions", lockCode)}},
	}
	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	api.EditMessageText(chatID, messageID, text, &keyboard)
}

func showLockConfirm(chatID string, messageID int, chatIDInt int64, lockCode string, userID int, sess LockSession) {
	lockName := lockCodeToName(lockCode)
	emoji := punishmentToEmoji(sess.NewPunish)
	text := fmt.Sprintf("⚖️ جریمهٔ «%s» برای قفل %s اعمال شود؟", emoji, lockName)
	buttons := [][]api.InlineKeyboardButton{
		{
			{Text: "فعالسازی", CallbackData: fmt.Sprintf("panel:lockstep:%s:apply", lockCode)},
			{Text: "بازگشت", CallbackData: fmt.Sprintf("panel:lockstep:%s:punish_select", lockCode)},
		},
	}
	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	api.EditMessageText(chatID, messageID, text, &keyboard)
}

// ============================================================
//  State Machine ضداسپم (spamstep)
// ============================================================
func handleSpamStep(chatID string, messageID int, chatIDInt int64, query *api.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		return
	}
	step := parts[2]
	userID := query.From.ID
	msgID := query.Message.MessageID

	switch step {
	case "punish_select":
		saveSpamSession(int64(userID), chatIDInt, SpamSession{
			ChatID: chatIDInt,
			Step:   "punish_select",
		})
		showSpamPunishSelect(chatID, msgID, chatIDInt, userID)
		api.AnswerCallback(query.ID, "", false)
	case "confirm":
		if len(parts) < 4 {
			return
		}
		punishment := parts[3]
		sess, ok := getSpamSession(int64(userID), chatIDInt)
		if !ok {
			api.AnswerCallback(query.ID, "⏱ جلسه منقضی شده، دوباره تلاش کنید.", true)
			return
		}
		sess.Step = "confirm"
		sess.NewPunish = punishment
		saveSpamSession(int64(userID), chatIDInt, sess)
		showSpamConfirm(chatID, msgID, chatIDInt, userID, sess)
		api.AnswerCallback(query.ID, "", false)
	case "apply":
		sess, ok := getSpamSession(int64(userID), chatIDInt)
		if !ok {
			api.AnswerCallback(query.ID, "⏱ جلسه منقضی شده، دوباره تلاش کنید.", true)
			return
		}
		database.DB.Exec("INSERT OR REPLACE INTO antispam_settings (chat_id, punishment) VALUES (?, ?)", chatIDInt, sess.NewPunish)
		services.InvalidateConfig(chatIDInt)
		api.AnswerCallback(query.ID, fmt.Sprintf("✅ جریمه اسپم به %s تنظیم شد", punishmentToEmoji(sess.NewPunish)), true)
		deleteSpamSession(int64(userID), chatIDInt)
	default:
		api.AnswerCallback(query.ID, "", false)
	}
}

func showSpamPunishSelect(chatID string, messageID int, chatIDInt int64, userID int) {
	cfg := services.GetAntispamConfig(chatIDInt)
	current := "🗑️ فقط حذف"
	switch cfg.Punishment {
	case "warn": current = "⚠️ اخطار"
	case "mute": current = "🔇 سکوت"
	case "ban": current = "⛔ بن"
	}
	text := fmt.Sprintf("⚖️ **جریمه اسپم**\n\nفعلی: %s\n\nجریمه جدید را انتخاب کنید:", current)
	buttons := [][]api.InlineKeyboardButton{
		{{Text: "🗑️ فقط حذف", CallbackData: "panel:spamstep:confirm:delete"}},
		{{Text: "⚠️ اخطار", CallbackData: "panel:spamstep:confirm:warn"}},
		{{Text: "🔇 سکوت", CallbackData: "panel:spamstep:confirm:mute"}},
		{{Text: "⛔ بن", CallbackData: "panel:spamstep:confirm:ban"}},
		{{Text: "🔙 بازگشت", CallbackData: fmt.Sprintf("panel:antispam:%d", messageID)}},
	}
	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	api.EditMessageText(chatID, messageID, text, &keyboard)
}

func showSpamConfirm(chatID string, messageID int, chatIDInt int64, userID int, sess SpamSession) {
	emoji := punishmentToEmoji(sess.NewPunish)
	text := fmt.Sprintf("⚖️ جریمهٔ «%s» برای اسپم اعمال شود؟", emoji)
	buttons := [][]api.InlineKeyboardButton{
		{
			{Text: "اعمال", CallbackData: "panel:spamstep:apply"},
			{Text: "بازگشت", CallbackData: "panel:spamstep:punish_select"},
		},
	}
	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	api.EditMessageText(chatID, messageID, text, &keyboard)
}
