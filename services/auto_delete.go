package services

import (
	"fmt"
	"sync"
	"time"

	"dalbot/api"
	"dalbot/logger"
)

type autoDeleteEntry struct {
	messageID int
	deleteAt  time.Time
}

var (
	autoDeleteStore = struct {
		sync.Mutex
		entries map[int64][]autoDeleteEntry // chatID -> entries
	}{entries: make(map[int64][]autoDeleteEntry)}
)

// AddAutoDelete schedules a bot message for future deletion.
func AddAutoDelete(chatID int64, messageID int, delay time.Duration) {
	autoDeleteStore.Lock()
	defer autoDeleteStore.Unlock()
	autoDeleteStore.entries[chatID] = append(autoDeleteStore.entries[chatID], autoDeleteEntry{
		messageID: messageID,
		deleteAt:  time.Now().Add(delay),
	})
	logger.Info.Printf("AutoDelete stored: chat=%d, msg=%d, delay=%v, total_entries=%d", chatID, messageID, delay, len(autoDeleteStore.entries[chatID]))
}

// StartAutoDeleteWorker runs periodically and removes messages whose time has come.
func StartAutoDeleteWorker() {
	logger.Info.Println("AutoDelete worker started")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error.Printf("AutoDelete worker panic: %v", r)
				// ری‌استارت worker در صورت پنیک
				time.Sleep(5 * time.Second)
				go StartAutoDeleteWorker()
			}
		}()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			autoDeleteStore.Lock()
			for chatID, entries := range autoDeleteStore.entries {
				var remaining []autoDeleteEntry
				for _, e := range entries {
					if now.After(e.deleteAt) {
						err := api.DeleteMessage(fmt.Sprintf("%d", chatID), e.messageID)
						if err != nil {
							logger.Error.Printf("AutoDelete failed for %d in %d: %v", e.messageID, chatID, err)
						} else {
							logger.Info.Printf("AutoDelete deleted: chat=%d, msg=%d", chatID, e.messageID)
						}
					} else {
						remaining = append(remaining, e)
					}
				}
				if len(remaining) == 0 {
					delete(autoDeleteStore.entries, chatID)
				} else {
					autoDeleteStore.entries[chatID] = remaining
				}
			}
			autoDeleteStore.Unlock()
		}
	}()
}
