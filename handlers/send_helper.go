package handlers

import (
	"time"

	"dalbot/api"
	"dalbot/logger"
)

const (
	AutoDel5  = 5 * time.Second
	AutoDel10 = 10 * time.Second
	AutoDel30 = 30 * time.Second
	AutoDel60 = 60 * time.Second
	AutoDel7  = 7 * time.Second
)

func sendGroupMsg(chatID string, delay time.Duration, text string) (*api.Message, error) {
	msg, err := api.SendMessage(chatID, text)
	if err != nil || msg == nil {
		return msg, err
	}
	if delay > 0 {
		chatIDInt := parseInt64Chat(chatID)
		scheduleAutoDelete(chatIDInt, msg.MessageID, delay)
		logger.Info.Printf("Auto-delete scheduled: chat=%d, msg=%d, delay=%v", chatIDInt, msg.MessageID, delay)
	}
	return msg, nil
}
