package services

import (
	"strings"
	"sync"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
)

var (
	locksCache   = make(map[int64][]string)
	locksCacheMu sync.Mutex
)

func InvalidateLocksCache(chatID int64) {
	locksCacheMu.Lock()
	delete(locksCache, chatID)
	locksCacheMu.Unlock()
}

func getActiveLocks(chatID int64) []string {
	locksCacheMu.Lock()
	cached, ok := locksCache[chatID]
	locksCacheMu.Unlock()
	if ok {
		return cached
	}

	rows, err := database.DB.Query("SELECT lock_type FROM locks WHERE chat_id = ? AND is_locked = 1", chatID)
	if err != nil {
		logger.Error.Printf("CheckContentLocks query error: %v", err)
		return nil
	}
	defer rows.Close()

	var locks []string
	for rows.Next() {
		var lt string
		if err := rows.Scan(&lt); err == nil {
			locks = append(locks, lt)
		}
	}

	locksCacheMu.Lock()
	locksCache[chatID] = locks
	locksCacheMu.Unlock()

	return locks
}

func CheckContentLocks(chatID int64, msg *api.Message) bool {
	for _, lockType := range getActiveLocks(chatID) {
		if ViolatesLock(lockType, msg) {
			return true
		}
	}
	return false
}

// ViolatesLock returns true if the message violates the given lock type.
func ViolatesLock(lockType string, msg *api.Message) bool {
	switch lockType {
	case "link":
		t := strings.ToLower(msg.Text)
		if strings.Contains(t, "http") || strings.Contains(t, "ble.ir") || strings.Contains(t, "t.me/") {
			return true
		}
	case "forward":
		if msg.ForwardFrom != nil || msg.ForwardFromChat != nil || msg.ForwardDate > 0 {
			return true
		}
	case "ad":
		adKeywords := []string{"تبلیغ", "خرید", "فروش", "تخفیف", "کلیک", "لینک", "عضویت"}
		for _, kw := range adKeywords {
			if strings.Contains(msg.Text, kw) {
				return true
			}
		}
	case "photo":
		if len(msg.Photo) > 0 {
			return true
		}
	case "video":
		if msg.Video != nil {
			return true
		}
	case "audio":
		if msg.Audio != nil || msg.Voice != nil {
			return true
		}
	case "longtext":
		if len(msg.Text) > 500 {
			return true
		}
	case "sticker":
		if msg.Sticker != nil {
			return true
		}
	case "gif":
		if msg.Animation != nil {
			return true
		}
	case "poll":
		if msg.Poll != nil {
			return true
		}
	case "all":
		return true
	}
	return false
}

func IsPollLocked(chatID int64) bool {
	for _, lockType := range getActiveLocks(chatID) {
		if lockType == "poll" || lockType == "all" {
			return true
		}
	}
	return false
}
