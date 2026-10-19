package handlers

import (
	"fmt"
	"strconv"
	"time"

	"dalbot/api"
	"dalbot/config"
	"dalbot/logger"
)

func handleNewMembers(chatID string, newMembers []api.User, chatTitle string) {
	for _, user := range newMembers {
		if user.ID == config.BotID {
			continue
		}

		firstName := user.FirstName
		message := fmt.Sprintf(
			"🌸 %s عزیز!\n\n"+
				"به گروه **%s** خوش آمدید.\n"+
				"امیدواریم در کنار هم لحظات خوبی را سپری کنیم.\n"+
				"🎉 به خانواده ما پیوستی!",
			"["+firstName+"](uid:"+strconv.Itoa(user.ID)+")",
			chatTitle,
		)

		sentMsg, err := api.SendMessage(chatID, message)
		if err != nil {
			logger.Error.Printf("handleNewMembers: failed to send welcome to %d: %v", user.ID, err)
			continue
		}

		// حذف خودکار پس از ۷ ثانیه
		go func(msgID int) {
			time.Sleep(7 * time.Second)
			api.DeleteMessage(chatID, msgID)
		}(sentMsg.MessageID)
	}
}
