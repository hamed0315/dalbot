package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	BotToken    string
	OwnerID     string
	BotID       int
	Channel1    string
	Channel2    string
	Channel3    string
	BaseURL     string
	MonitorMode bool
	DebugMode   bool
)

func Load() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system env")
	}

	BotToken = os.Getenv("BOT_TOKEN")
	OwnerID  = os.Getenv("OWNER_ID")
	Channel1 = os.Getenv("CHANNEL_1")
	Channel2 = os.Getenv("CHANNEL_2")
	Channel3 = os.Getenv("CHANNEL_3")

	if BotToken == "" || OwnerID == "" {
		log.Fatal("BOT_TOKEN and OWNER_ID must be set")
	}

	BaseURL = "https://tapi.bale.ai/bot" + BotToken

	MonitorMode = os.Getenv("MONITOR") == "true"
	DebugMode   = os.Getenv("DEBUG") == "true"
}
