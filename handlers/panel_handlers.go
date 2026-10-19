package handlers

import (
	"strconv"
	"strings"

	"dalbot/api"
	"dalbot/models"
)

func handlePanelCallback(query *api.CallbackQuery) {
	data := query.Data
	chatID := strconv.Itoa(query.Message.Chat.ID)
	chatIDInt := parseInt64Chat(chatID)
	msgID := query.Message.MessageID
	parts := strings.Split(data, ":")

	if len(parts) < 2 {
		return
	}

	switch parts[1] {
	case "close":
		api.DeleteMessage(chatID, msgID)
		return

	case "main":
		_, _ = models.GetGroup(chatIDInt)
		showMainPanel(chatID, msgID, query.From.FirstName, chatIDInt, false)

	case "lockstep":
		if len(parts) >= 3 {
			handleLockStep(chatID, msgID, chatIDInt, query, parts)
		}
		return

	case "spamstep":
		handleSpamStep(chatID, msgID, chatIDInt, query, parts)
		return

	case "locks":
		if len(parts) >= 3 {
			switch parts[2] {
			case "main":
				showLockList(chatID, msgID, chatIDInt)
			case "help":
				api.AnswerCallback(query.ID, "برای مدیریت قفل روی دکمه آن کلیک کنید.", true)
			case "actions":
				if len(parts) >= 4 {
					lockCode := parts[3]
					showLockActionsNew(chatID, msgID, chatIDInt, lockCode, query.From.ID)
				}
			}
		}
		return



	case "users":
		if len(parts) >= 3 && parts[2] == "clearwarns" {
			promptClearWarns(chatID, msgID)
			api.AnswerCallback(query.ID, "", false)
			return
		}
		showUsersPanel(chatID, msgID)
		api.AnswerCallback(query.ID, "", false)
		return

	case "forced_join":
		if len(parts) >= 3 {
			action := parts[2]
			switch action {
			case "add":
				conversationState[int64(query.From.ID)] = "awaiting_forced_channel_add"
				api.SendMessage(chatID, "📝 نام کانال را بفرستید (@test):")
				api.AnswerCallback(query.ID, "", false)
				return
			case "remove":
				conversationState[int64(query.From.ID)] = "awaiting_forced_channel_remove"
				api.SendMessage(chatID, "📝 نام کانال را بفرستید:")
				api.AnswerCallback(query.ID, "", false)
				return
			}
		}
		showForcedJoinPanel(chatID, msgID, chatIDInt)
		api.AnswerCallback(query.ID, "", false)
		return

	case "status":
		showGroupStatusPanel(chatID, msgID, chatIDInt)

	case "settings":
		if len(parts) >= 3 {
			action := parts[2]
			switch action {
			case "rules":
				conversationState[int64(query.From.ID)] = "awaiting_group_rules"
				api.SendMessage(chatID, "📝 متن قوانین را بفرستید:")
				api.AnswerCallback(query.ID, "", false)
				return
			case "welcome":
				conversationState[int64(query.From.ID)] = "awaiting_welcome_message"
				api.SendMessage(chatID, "📝 متن خوش‌آمدگویی را بفرستید:")
				api.AnswerCallback(query.ID, "", false)
				return
			}
		}
		showGroupSettingsPanel(chatID, msgID)
		api.AnswerCallback(query.ID, "", false)
		return

	case "quickpurge":
		handlePurge(chatID, query.From.ID, 0, "", "", []string{"100"}, 0, 0)
		api.AnswerCallback(query.ID, "🧹 پاکسازی ۱۰۰ پیام ثبت شد.", false)
		return

	case "help":
		api.AnswerCallback(query.ID, "به پنل مدیریت دال خوش آمدید!\nبرای دیدن قفل‌ها روی «قفل‌ها و جریمه‌ها» بزنید.", true)
		return
	}

	api.AnswerCallback(query.ID, "", false)
}

func promptClearWarns(chatID string, messageID int) {
	text := "🧹 **حذف اخطارها**\n\nروی پیام کاربر ریپلای بزنید و سپس روی دکمه کلیک کنید."
	buttons := [][]api.InlineKeyboardButton{
		{{Text: "🔙 بازگشت", CallbackData: "panel:users"}},
	}
	keyboard := api.InlineKeyboardMarkup{InlineKeyboard: buttons}
	api.EditMessageText(chatID, messageID, text, &keyboard)
}
