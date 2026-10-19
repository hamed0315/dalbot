package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"dalbot/api"
	"dalbot/config"
	"dalbot/database"
	"dalbot/logger"
	"dalbot/models"
	"dalbot/services"
)

type CommandFunc func(chatID string, userID int, targetUserID int, targetFirstName string, adminFirstName string, args []string, replyToID int, messageID int)

var commandMap = map[string]CommandFunc{
	"راهنما":            handleHelp,
	"قفل":             handleLock,
	"باز":             handleUnlock,
	".اخطار":           handleWarn,
	".بن":              handleBan,
	".کیک":             handleBan,
	".صیک":             handleBan,
	".سکوت":            handleMute,
	".انبن":            handleUnban,
	".رفع سکوت":        handleUnmute,
	".حذف اخطار":       handleClearWarns,
	"ثبت":             handleSetBio,
	"ثبت اصل":           handleSetBio,
	"اصل":             handleGetBio,
	"معرفی":           handleGetBio,
	"لینک":            handleInviteMenu,
	"ترفیع":           handlePromote,
	"تنزیل":           handleDemote,
	"ویژه":            handleSpecial,
	"فیلتر":           handleAddBadWord,
	"رنکینگ":          handleRanking,
	"آمار":            handleStats,
	".پین":             handlePin,
	".حذف پین":         handleUnpin,
	"پاکسازی":         handlePurge,
	"ساخت دعوت":       handleCreateInviteLink,
	"تگ":              handleTagMenu,
	"پنل":             handlePanel,
}

var conversationState = map[int64]string{}

var (
	spamMutex       sync.Mutex
	lastMsgTime     = make(map[int64]time.Time)
	lastMsgText     = make(map[int64]string)
	dupCount        = make(map[int64]int)
	floodTimestamps = make(map[int64][]time.Time)
)

const (
	duplicateWindow = 2 * time.Second
	maxDuplicates   = 2
	floodWindow     = 3 * time.Second
	maxFloodMsgs    = 5
)

func isSpamFast(chatID int64, userID int, text string) bool {
	spamMutex.Lock()
	defer spamMutex.Unlock()

	now := time.Now()
	if len(strings.TrimSpace(text)) == 0 {
		return false
	}

	lastTime, exists := lastMsgTime[chatID]
	lastText := lastMsgText[chatID]
	cnt := dupCount[chatID]

	if exists && text == lastText && now.Sub(lastTime) <= duplicateWindow {
		cnt++
		dupCount[chatID] = cnt
		if cnt >= maxDuplicates {
			return true
		}
	} else {
		dupCount[chatID] = 0
	}
	lastMsgTime[chatID] = now
	lastMsgText[chatID] = text

	ts := floodTimestamps[chatID]
	ts = append(ts, now)
	var recent []time.Time
	for _, t := range ts {
		if now.Sub(t) <= floodWindow {
			recent = append(recent, t)
		}
	}
	floodTimestamps[chatID] = recent
	if len(recent) > maxFloodMsgs {
		return true
	}

	return false
}

func ProcessUpdate(upd *api.Update) {
	if upd.Message == nil && upd.CallbackQuery == nil && upd.Poll == nil {
		return
	}

	if upd.CallbackQuery != nil {
		handleCallback(upd.CallbackQuery)
		return
	}

	if upd.Poll != nil && upd.Message == nil && upd.CallbackQuery == nil {
		return
	}

	msg := upd.Message
	if msg == nil {
		return
	}

	if time.Since(time.Unix(int64(msg.Date), 0)) > 60*time.Second {
		return
	}

	chatIDStr := strconv.Itoa(msg.Chat.ID)
	userID := 0
	adminFirstName := ""
	if msg.From != nil {
		userID = msg.From.ID
		adminFirstName = msg.From.FirstName
	}
	text := strings.TrimSpace(msg.Text)

	if msg.Chat.Type == "private" {
		if msg.From != nil {
			processPrivate(msg)
		}
		return
	}

	// ---- group ----

	if msg.From == nil {
		if msg.Poll != nil && services.IsPollLocked(parseInt64(chatIDStr)) {
			if err := api.DeleteMessage(chatIDStr, msg.MessageID); err != nil {
				logger.Error.Printf("safeDeleteMessage for anonymous poll failed: %v", err)
			}
			return
		}
		if msg.ForwardDate > 0 || msg.ForwardFrom != nil || msg.ForwardFromChat != nil {
			if services.CheckContentLocks(parseInt64(chatIDStr), msg) {
				if err := api.DeleteMessage(chatIDStr, msg.MessageID); err != nil {
					logger.Error.Printf("safeDeleteMessage for forward lock (anonymous) failed: %v", err)
				}
			}
		}
		return
	}

	isPinnedMsg := msg.PinnedMessage != nil
	isFromBot := msg.From.ID == config.BotID

	if !isFromBot && !isPinnedMsg {
		services.IncrementMessageCount(int64(msg.Chat.ID), userID)
		services.RecordUserActivity(int64(msg.Chat.ID), userID, msg.From.Username, msg.From.FirstName)
	}

	if !isFromBot && isSpamFast(int64(msg.Chat.ID), userID, text) {
		return
	}

	if !isFromBot && !isPinnedMsg && msg.MessageID > 0 {
		if msg.PinnedMessage != nil && msg.PinnedMessage.MessageID > 0 {
			services.RemovePendingMessage(int64(msg.Chat.ID), msg.PinnedMessage.MessageID)
		}
		services.AddPendingMessage(int64(msg.Chat.ID), msg.MessageID)
		services.AddUserMessageToPurge(int64(msg.Chat.ID), msg.MessageID)
	}

	if isUserGloballyBanned(int64(userID)) {
		level := getUserLevel(int64(msg.Chat.ID), userID)
		if level == "regular" || level == "special" {
			if err := api.DeleteMessage(chatIDStr, msg.MessageID); err != nil {
				logger.Error.Printf("safeDeleteMessage for globally banned failed: %v", err)
			}
			api.BanChatMember(chatIDStr, userID)
			api.SendMessage(chatIDStr, "⛔ " + formatMention(userID, "", msg.From.FirstName) + " به دلیل سابقه اسپم در چندین گروه، از این گروه نیز اخراج شد.")
			title := msg.Chat.Title
			if title == "" {
				title = fmt.Sprintf("گروه %d", msg.Chat.ID)
			}
			services.LogToOwner("CRITICAL", fmt.Sprintf("⛔ %s (ID: %d) به دلیل اسپم در چندین گروه، از گروه %s (ID: %d) اخراج سراسری شد.", formatMention(userID, "", msg.From.FirstName), userID, title, msg.Chat.ID))
			return
		}
	}

	// 🆕 مدیریت گزارش (همه می‌توانند بدون محدودیت سطح استفاده کنند)
	if text == "گزارش" || strings.HasPrefix(text, "گزارش ") {
		targetUserID := 0
		targetFirstName := ""
		if msg.ReplyTo != nil {
			if msg.ReplyTo.From != nil {
				targetUserID = msg.ReplyTo.From.ID
				targetFirstName = msg.ReplyTo.From.FirstName
			}
		}
		handleReport(chatIDStr, userID, targetUserID, targetFirstName, msg.From.FirstName, strings.Fields(text), msg.ReplyTo, msg.MessageID)
		return
	}

	level := getUserLevel(int64(msg.Chat.ID), userID)

	if strings.HasPrefix(text, "نصب") {
		handleInstall(chatIDStr, userID, 0, "", adminFirstName, nil, 0, msg.MessageID)
		return
	}

	if text == "تگ" {
		if level == "regular" || level == "special" {
			return
		}
		handleTagMenu(chatIDStr, userID, 0, "", adminFirstName, nil, 0, msg.MessageID)
		return
	}

	if text == "لینک" {
		if level == "regular" || level == "special" {
			return
		}
		handleInviteMenu(chatIDStr, userID, 0, "", adminFirstName, nil, 0, msg.MessageID)
		return
	}

	if text == "پنل" {
		if level == "regular" || level == "special" {
			return
		}
		handlePanel(chatIDStr, userID, 0, "", adminFirstName, nil, 0, msg.MessageID)
		return
	}

	targetUserID := 0
	targetFirstName := ""
	if msg.ReplyTo != nil {
		if msg.ReplyTo.From != nil {
			targetUserID = msg.ReplyTo.From.ID
			targetFirstName = msg.ReplyTo.From.FirstName
		}
	}

	if strings.HasPrefix(text, "عضویت اجباری") {
		if level == "regular" || level == "special" {
			return
		}
		args := strings.TrimSpace(strings.TrimPrefix(text, "عضویت اجباری"))
		handleAddForcedChannel(chatIDStr, userID, 0, "", adminFirstName, strings.Fields(args), 0, msg.MessageID)
		return
	}
	if strings.HasPrefix(text, "حذف عضویت اجباری") {
		if level == "regular" || level == "special" {
			return
		}
		args := strings.TrimSpace(strings.TrimPrefix(text, "حذف عضویت اجباری"))
		handleRemoveForcedChannel(chatIDStr, userID, 0, "", adminFirstName, strings.Fields(args), 0, msg.MessageID)
		return
	}
	if strings.HasPrefix(text, "لیست عضویت اجباری") {
		if level == "regular" || level == "special" {
			return
		}
		handleListForcedChannels(chatIDStr, userID, 0, "", adminFirstName, nil, 0, msg.MessageID)
		return
	}

	if strings.HasPrefix(text, "حذف فیلتر") {
		if level == "regular" || level == "special" {
			return
		}
		args := strings.TrimSpace(strings.TrimPrefix(text, "حذف فیلتر"))
		handleRemoveBadWord(chatIDStr, userID, 0, "", adminFirstName, strings.Fields(args), 0, msg.MessageID)
		return
	}
	if strings.HasPrefix(text, "لیست فیلتر") {
		if level == "regular" || level == "special" {
			return
		}
		handleListBadWords(chatIDStr, userID, 0, "", adminFirstName, nil, 0, msg.MessageID)
		return
	}

	if strings.HasPrefix(text, ".رفع سکوت") && text != ".رفع سکوت همه" {
		if level == "regular" || level == "special" {
			return
		}
		args := strings.TrimSpace(strings.TrimPrefix(text, ".رفع سکوت"))
		handleUnmute(chatIDStr, userID, targetUserID, targetFirstName, adminFirstName, strings.Fields(args), 0, msg.MessageID)
		return
	}
	if text == ".رفع سکوت همه" {
		if level == "regular" || level == "special" {
			return
		}
		handleClearMutes(chatIDStr, userID, 0, "", adminFirstName, nil, 0, msg.MessageID)
		return
	}
	if strings.HasPrefix(text, ".حذف اخطار") {
		if level == "regular" || level == "special" {
			return
		}
		args := strings.TrimSpace(strings.TrimPrefix(text, ".حذف اخطار"))
		handleClearWarns(chatIDStr, userID, targetUserID, targetFirstName, adminFirstName, strings.Fields(args), 0, msg.MessageID)
		return
	}
	if strings.HasPrefix(text, ".انبن") {
		if level == "regular" || level == "special" {
			return
		}
		args := strings.TrimSpace(strings.TrimPrefix(text, ".انبن"))
		handleUnban(chatIDStr, userID, targetUserID, targetFirstName, adminFirstName, strings.Fields(args), 0, msg.MessageID)
		return
	}

	if strings.HasPrefix(text, "تنظیم اسپم سیل") {
		if level == "regular" || level == "special" {
			return
		}
		args := strings.TrimSpace(strings.TrimPrefix(text, "تنظیم اسپم سیل"))
		handleAntispamFlood(chatIDStr, userID, 0, "", adminFirstName, strings.Fields(args), 0, msg.MessageID)
		return
	}
	if strings.HasPrefix(text, "تنظیم اسپم تکرار") {
		if level == "regular" || level == "special" {
			return
		}
		args := strings.TrimSpace(strings.TrimPrefix(text, "تنظیم اسپم تکرار"))
		handleAntispamDuplicate(chatIDStr, userID, 0, "", adminFirstName, strings.Fields(args), 0, msg.MessageID)
		return
	}
	if strings.HasPrefix(text, "تنظیم اسپم سکوت") {
		if level == "regular" || level == "special" {
			return
		}
		args := strings.TrimSpace(strings.TrimPrefix(text, "تنظیم اسپم سکوت"))
		handleAntispamMute(chatIDStr, userID, 0, "", adminFirstName, strings.Fields(args), 0, msg.MessageID)
		return
	}
	if strings.HasPrefix(text, "تنظیم اسپم اخراج") {
		if level == "regular" || level == "special" {
			return
		}
		handleAntispamBan(chatIDStr, userID, 0, "", adminFirstName, nil, 0, msg.MessageID)
		return
	}

	if strings.HasPrefix(text, "ساخت دعوت") {
		if level == "regular" || level == "special" {
			return
		}
		handleCreateInviteLink(chatIDStr, userID, 0, "", adminFirstName, nil, 0, msg.MessageID)
		return
	}

	state, inConversation := conversationState[int64(userID)]
	if inConversation {
		if state == "awaiting_forced_channel_add" {
			if level == "regular" || level == "special" {
				return
			}
			handleAddForcedChannel(chatIDStr, userID, 0, "", adminFirstName, strings.Fields(text), 0, msg.MessageID)
			delete(conversationState, int64(userID))
			return
		} else if state == "awaiting_forced_channel_remove" {
			if level == "regular" || level == "special" {
				return
			}
			handleRemoveForcedChannel(chatIDStr, userID, 0, "", adminFirstName, strings.Fields(text), 0, msg.MessageID)
			delete(conversationState, int64(userID))
			return
		}
	}

	group, err := models.GetGroup(int64(msg.Chat.ID))
	if err != nil || group == nil || !group.IsActive {
		return
	}

	if !isFromBot {
		missingChannels := services.CheckAllMissingChannels(int64(msg.Chat.ID), userID)
		if len(missingChannels) > 0 {
			if err := api.DeleteMessage(chatIDStr, msg.MessageID); err != nil {
				logger.Error.Printf("safeDeleteMessage for forced join failed: %v", err)
			}
			sendForcedJoinMessage(chatIDStr, userID, msg.From.FirstName, missingChannels)
			return
		}
	}

	// ============ تمام هوش مصنوعی و فیلترهای محتوایی پیشرفته در این نسخه غیرفعال شده‌اند ============

	// فقط اسپم فنی (Flood / Duplicate) و قفل‌های محتوا (لینک، فوروارد، نظرسنجی و ...) فعال هستند.
	if !isFromBot && level != "super_owner" && level != "owner" && level != "senior_admin" && level != "admin" {
		if services.ContainsBadWord(int64(msg.Chat.ID), text) {
			if err := api.DeleteMessage(chatIDStr, msg.MessageID); err != nil {
				logger.Error.Printf("safeDeleteMessage for badword failed: %v", err)
			}
			notifyDeletion(chatIDStr, userID, msg.From.FirstName, "کلمه ممنوع")
			return
		}
	}

	if !isFromBot && level != "super_owner" && level != "owner" && level != "senior_admin" && level != "admin" && level != "special" {
		if services.CheckContentLocks(int64(msg.Chat.ID), msg) {
			if err := api.DeleteMessage(chatIDStr, msg.MessageID); err != nil {
				logger.Error.Printf("safeDeleteMessage for content lock failed: %v", err)
			}
			applyLockPunishment(int64(msg.Chat.ID), userID, msg.From.FirstName, msg)
			return
		}
	}

	// ============ پایان بخش غیرفعال‌سازی ============

	if text == "" {
		return
	}

	if level == "regular" || level == "special" {
		return
	}

	parts := strings.Fields(text)
	cmd := parts[0]
	args := parts[1:]

	needsDot := map[string]bool{"بن": true, "کیک": true, "صیک": true, "سکوت": true, "اخطار": true, "انبن": true, "رفع سکوت": true, "حذف اخطار": true, "پین": true, "حذف پین": true}
	if needsDot[cmd] {
		return
	}

	replyToID := 0
	if msg.ReplyTo != nil {
		replyToID = msg.ReplyTo.MessageID
		if msg.ReplyTo.From != nil {
			targetUserID = msg.ReplyTo.From.ID
			targetFirstName = msg.ReplyTo.From.FirstName
			if cmd == "ثبت" || cmd == "ثبت اصل" {
				if msg.ReplyTo.Text != "" {
					args = []string{msg.ReplyTo.Text}
				} else {
					args = []string{""}
				}
			}
			if cmd == "تنظیم قوانین" {
				if msg.ReplyTo.Text != "" {
					args = []string{msg.ReplyTo.Text}
				} else {
					args = []string{""}
				}
			}
		}
	}

	if cmd == "قوانین" {
		handleShowRules(chatIDStr, userID, targetUserID, targetFirstName, adminFirstName, args, 0, msg.MessageID)
		return
	}
	if cmd == "رنکینگ" {
		handleRanking(chatIDStr, userID, targetUserID, targetFirstName, adminFirstName, args, 0, msg.MessageID)
		return
	}

	handler, exists := commandMap[cmd]
	mayExec := mayExecute(level, cmd)
	if exists && mayExec {
		handler(chatIDStr, userID, targetUserID, targetFirstName, adminFirstName, args, replyToID, msg.MessageID)
		if level == "super_owner" || level == "owner" || level == "senior_admin" {
			go services.CheckAndUpdateGroupStatus(parseInt64(chatIDStr), group.OwnerID)
		}
		if config.MonitorMode {
			logger.Monitor.Printf("Chat:%s | User:%d | Level:%s | Command:%s | Target:%d", chatIDStr, userID, level, cmd, targetUserID)
		}
	}
}

// ======================== توابع کمکی ========================

func formatMention(userID int, username, firstName string) string {
	displayName := fmt.Sprintf("کاربر %d", userID)
	if username != "" {
		displayName = "@" + username
	} else if firstName != "" {
		displayName = firstName
	}
	return fmt.Sprintf("[%s](uid:%d)", displayName, userID)
}

func notifyDeletion(chatID string, userID int, firstName string, reason string) {
	msg := fmt.Sprintf("⛔ پیام %s به دلیل «%s» حذف شد.", formatMention(userID, "", firstName), reason)
	sentMsg, err := api.SendMessage(chatID, msg)
	if err == nil && sentMsg != nil {
		chatIDInt := parseInt64Chat(chatID)
		scheduleAutoDelete(chatIDInt, sentMsg.MessageID, 30*time.Second)
	}
}

func applyLockPunishment(chatID int64, userID int, firstName string, msg *api.Message) {
	lockType := detectViolatedLock(chatID, msg)
	if lockType == "" {
		return
	}
	lp := services.GetLockPunishment(chatID, lockType)
	chatIDStr := strconv.FormatInt(chatID, 10)

	lockName := "نامشخص"
	switch lockType {
	case "link": lockName = "لینک"
	case "forward": lockName = "فوروارد"
	case "ad": lockName = "تبلیغات"
	case "photo": lockName = "عکس"
	case "video": lockName = "ویدئو"
	case "audio": lockName = "صوت"
	case "longtext": lockName = "طولانی"
	case "sticker": lockName = "استیکر"
	case "gif": lockName = "گیف"
	case "poll": lockName = "نظرسنجی"
	case "all": lockName = "گروه"
	}
	notifyDeletion(chatIDStr, userID, firstName, "نقض قفل "+lockName)

	if lockType == "all" {
		return
	}
	switch lp.Punishment {
	case "warn":
		warnUser(chatID, userID, 0, "نقض قفل "+lockType)
	case "mute":
		api.RestrictChatMember(chatIDStr, userID, false)
		unmuteAt := time.Now().Add(time.Duration(lp.MuteDurationHrs) * time.Hour).UTC().Format("2006-01-02 15:04:05")
		database.DB.Exec("INSERT OR REPLACE INTO mutes (chat_id, user_id, unmute_at) VALUES (?, ?, ?)", chatID, userID, unmuteAt)
		api.SendMessage(chatIDStr, fmt.Sprintf("🔇 %s به دلیل نقض قفل %s سکوت شد.", formatMention(userID, "", firstName), lockType))
	case "ban":
		api.BanChatMember(chatIDStr, userID)
		api.SendMessage(chatIDStr, fmt.Sprintf("⛔ %s به دلیل نقض قفل %s اخراج شد.", formatMention(userID, "", firstName), lockType))
	}
}

func detectViolatedLock(chatID int64, msg *api.Message) string {
	locks := []string{"link", "forward", "ad", "photo", "video", "audio", "longtext", "sticker", "gif", "poll", "all"}
	for _, lockType := range locks {
		if services.ViolatesLock(lockType, msg) {
			return lockType
		}
	}
	return ""
}

func getUserLevel(chatID int64, userID int) string {
	if strconv.Itoa(userID) == config.OwnerID {
		return "super_owner"
	}
	group, _ := models.GetGroup(chatID)
	if group != nil && int64(userID) == group.OwnerID {
		return "owner"
	}
	adminLevel, err := models.GetAdminLevel(chatID, int64(userID))
	if err == nil {
		return string(adminLevel)
	}
	return "regular"
}

func mayExecute(userLevel string, command string) bool {
	if userLevel == "super_owner" || userLevel == "owner" || userLevel == "senior_admin" {
		return true
	}
	if userLevel == "admin" {
		if command == ".بن" || command == ".کیک" || command == ".صیک" || command == ".اخطار" ||
			command == "ترفیع" || command == "تنزیل" || command == "ویژه" ||
			command == "تنظیم قوانین" || command == ".رفع سکوت همه" ||
			command == ".پین" || command == ".حذف پین" ||
			command == "ساخت دعوت" || command == ".انبن" {
			return false
		}
		return true
	}
	return false
}

func applySpamPunishment(chatID string, userID int, firstName string) {
	cfg := services.GetAntispamConfig(parseInt64(chatID))
	if cfg.Punishment == "ban" {
		if err := api.BanChatMember(chatID, userID); err != nil {
			return
		}
		recordBan(parseInt64(chatID), int64(userID), firstName, parseOwnerID(), "سیستم", "اسپم (سیستم)")
		msg := fmt.Sprintf("⛔ %s به دلیل اسپم توسط سیستم اخراج شد.", formatMention(userID, "", firstName))
		api.SendMessage(chatID, msg)
	} else {
		if err := api.RestrictChatMember(chatID, userID, false); err != nil {
			api.SendMessage(chatID, "⚠️ سیستم نتوانست کاربر را سکوت کند.")
			return
		}
		chatIDInt := parseInt64(chatID)
		unmuteAt := time.Now().Add(time.Duration(cfg.MuteDurationHrs) * time.Hour).UTC().Format("2006-01-02 15:04:05")
		database.DB.Exec("INSERT OR REPLACE INTO mutes (chat_id, user_id, unmute_at) VALUES (?, ?, ?)", chatIDInt, userID, unmuteAt)
		msg := fmt.Sprintf("🔇 %s به دلیل اسپم توسط سیستم به مدت %d ساعت سکوت شد.", formatMention(userID, "", firstName), cfg.MuteDurationHrs)
		api.SendMessage(chatID, msg)
	}
}

func parseInt64(s string) int64 {
	var id int64
	fmt.Sscanf(s, "%d", &id)
	return id
}

func parseOwnerID() int64 {
	id, _ := strconv.ParseInt(config.OwnerID, 10, 64)
	return id
}

func isUserGloballyBanned(userID int64) bool {
	var count int
	database.DB.QueryRow("SELECT group_count FROM global_bans WHERE user_id = ?", userID).Scan(&count)
	return count >= 6
}

func isUserApproved(chatID int64, userID int) bool {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM approved_users WHERE chat_id = ? AND user_id = ?", chatID, userID).Scan(&count)
	return count > 0
}

func checkForcedJoinAsync(msg *api.Message, userID int, firstName string) {
	// دیگر استفاده نمی‌شود.
}

// ======================== دستور گزارش (بدون نادیده گرفتن گزارش‌دهنده) ========================

func handleReport(chatID string, reporterID int, targetUserID int, targetFirstName string, reporterFirstName string, args []string, replyToMsg *api.Message, messageID int) {
	if replyToMsg == nil {
		api.SendMessage(chatID, "⚠️ لطفاً روی پیام مورد نظر ریپلای بزنید و سپس دستور گزارش را ارسال کنید.")
		return
	}

	groupTitle := "گروه"
	chatIDInt := parseInt64Chat(chatID)
	group, _ := models.GetGroup(chatIDInt)
	if group != nil {
		groupTitle = group.Title
	}

	reportedContent := replyToMsg.Text
	if reportedContent == "" {
		if replyToMsg.Caption != "" {
			reportedContent = replyToMsg.Caption
		} else {
			reportedContent = " (پیام چندرسانه‌ای یا بدون متن) "
		}
	}

	reportText := fmt.Sprintf(
		"📢 **گزارش از گروه %s**\n👤 گزارش‌دهنده: %s\n🎯 کاربر گزارش‌شده: %s\n💬 متن پیام: %s",
		groupTitle,
		formatMention(reporterID, "", reporterFirstName),
		formatMention(targetUserID, "", targetFirstName),
		reportedContent,
	)

	admins := getGroupAdmins(chatIDInt)
	if len(admins) == 0 {
		api.SendMessage(chatID, "❌ هیچ ادمینی برای دریافت گزارش یافت نشد.")
		return
	}

	successCount := 0
	for _, adminID := range admins {
		// 🔥 حذف شرط نادیده گرفتن گزارش‌دهنده
		_, err := api.SendMessage(strconv.FormatInt(adminID, 10), reportText)
		if err != nil {
			logger.Error.Printf("handleReport: failed to send to admin %d: %v", adminID, err)
		} else {
			successCount++
		}
	}

	if successCount > 0 {
		api.SendMessage(chatID, "✅ گزارش شما با موفقیت به ادمین‌ها ارسال شد.")
	} else {
		// Fallback: ارسال گزارش در خود گروه
		api.SendMessage(chatID, "⚠️ ارسال خصوصی به ادمین‌ها ممکن نشد (احتمالاً ربات را استارت نکرده‌اند).\n📢 گزارش در همین گروه قرار می‌گیرد:")
		api.SendMessage(chatID, reportText)
	}
}

func getGroupAdmins(chatID int64) []int64 {
	var admins []int64
	group, _ := models.GetGroup(chatID)
	if group != nil {
		admins = append(admins, group.OwnerID)
	}
	rows, err := database.DB.Query("SELECT user_id FROM group_admins WHERE chat_id = ? AND level IN ('senior_admin', 'admin')", chatID)
	if err != nil {
		return admins
	}
	defer rows.Close()
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err == nil {
			already := false
			for _, a := range admins {
				if a == uid {
					already = true
					break
				}
			}
			if !already {
				admins = append(admins, uid)
			}
		}
	}
	return admins
}

func sendToGroupAdmins(chatID int64, exceptUserID int64, message string) {
	admins := getGroupAdmins(chatID)
	for _, adminID := range admins {
		if adminID != exceptUserID {
			api.SendMessage(strconv.FormatInt(adminID, 10), message)
		}
	}
}
