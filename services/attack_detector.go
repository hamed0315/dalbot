package services

import (
	"sync"
	"time"
)

// violationTracker tracks per-lock violations for each chat
type violationTracker struct {
	mu      sync.Mutex
	entries map[int64]map[string]*violationWindow
}

type violationWindow struct {
	mu         sync.Mutex
	timestamps []time.Time
}

var (
	atkTracker = &violationTracker{
		entries: make(map[int64]map[string]*violationWindow),
	}

	attackWindow    = 60 * time.Second
	attackThreshold = 15
)

func init() {
	go atkTracker.cleanup(1 * time.Minute)
}

// RecordLockViolation registers a violation of a specific lock type in a chat.
func RecordLockViolation(chatID int64, lockType string) {
	atkTracker.getWindow(chatID, lockType).add(time.Now())
}

// DetectAttack returns the list of lock types currently under attack.
func DetectAttack(chatID int64) []string {
	atkTracker.mu.Lock()
	defer atkTracker.mu.Unlock()

	chatMap, ok := atkTracker.entries[chatID]
	if !ok {
		return nil
	}

	var attacked []string
	cutoff := time.Now().Add(-attackWindow)
	for lockType, win := range chatMap {
		win.mu.Lock()
		count := 0
		for _, t := range win.timestamps {
			if t.After(cutoff) {
				count++
			}
		}
		win.mu.Unlock()
		if count >= attackThreshold {
			attacked = append(attacked, lockType)
		}
	}
	return attacked
}

// IsUnderAttack returns true if the given lock type is currently under attack in the chat.
func IsUnderAttack(chatID int64, lockType string) bool {
	win := atkTracker.getWindow(chatID, lockType)
	win.mu.Lock()
	defer win.mu.Unlock()
	cutoff := time.Now().Add(-attackWindow)
	count := 0
	for _, t := range win.timestamps {
		if t.After(cutoff) {
			count++
		}
	}
	return count >= attackThreshold
}

func (vt *violationTracker) getWindow(chatID int64, lockType string) *violationWindow {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	if _, ok := vt.entries[chatID]; !ok {
		vt.entries[chatID] = make(map[string]*violationWindow)
	}
	win, ok := vt.entries[chatID][lockType]
	if !ok {
		win = &violationWindow{}
		vt.entries[chatID][lockType] = win
	}
	return win
}

func (vw *violationWindow) add(t time.Time) {
	vw.mu.Lock()
	defer vw.mu.Unlock()
	vw.timestamps = append(vw.timestamps, t)
}

func (vt *violationTracker) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		vt.mu.Lock()
		cutoff := time.Now().Add(-attackWindow)
		for chatID, chatMap := range vt.entries {
			for lockType, win := range chatMap {
				win.mu.Lock()
				var fresh []time.Time
				for _, t := range win.timestamps {
					if t.After(cutoff) {
						fresh = append(fresh, t)
					}
				}
				win.timestamps = fresh
				if len(fresh) == 0 {
					delete(chatMap, lockType)
				}
				win.mu.Unlock()
			}
			if len(chatMap) == 0 {
				delete(vt.entries, chatID)
			}
		}
		vt.mu.Unlock()
	}
}
