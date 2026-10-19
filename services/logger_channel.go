package services

import (
	"fmt"
	"sync"
	"time"

	"dalbot/api"
	"dalbot/config"
	"dalbot/logger"
)

var (
	lastLogTime time.Time
	logMutex    sync.Mutex
	logCount    int
)

// LogToOwner sends a critical/warning/info message to the bot owner.
// level: "CRITICAL", "WARNING", "INFO"
func LogToOwner(level string, message string) {
	logMutex.Lock()
	defer logMutex.Unlock()

	// جلوگیری از ارسال بیش از ۵ پیام در ۱۰ ثانیه
	now := time.Now()
	if now.Sub(lastLogTime) < 10*time.Second {
		logCount++
		if logCount > 5 {
			return
		}
	} else {
		logCount = 0
	}
	lastLogTime = now

	prefix := ""
	switch level {
	case "CRITICAL":
		prefix = "🔴 "
	case "WARNING":
		prefix = "🟠 "
	case "INFO":
		prefix = "🔵 "
	default:
		prefix = "ℹ️ "
	}

	text := fmt.Sprintf("%s%s", prefix, message)
	_, err := api.SendMessage(config.OwnerID, text)
	if err != nil {
		logger.Error.Printf("LogToOwner failed: %v", err)
	}
}
