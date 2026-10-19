package services

import (
	"sync"
	"time"
)

type userReputation struct {
	mu           sync.Mutex
	score        int
	lastActivity time.Time
}

var (
	reputations = make(map[int64]map[int]*userReputation) // chatID -> userID -> reputation
	repMu       sync.RWMutex
)

const (
	initialScore    = 30
	maxScore        = 100
	minScore        = 0
	decayInterval   = 1 * time.Hour
	decayAmount     = -1 // کاهش امتیاز (هرچه کمتر = بهتر)
	likeBonus       = -2
	lockViolation   = 5
	floodPenalty    = 15
	banHistoryBonus = 20
)

func GetUserScore(chatID int64, userID int) int {
	repMu.RLock()
	defer repMu.RUnlock()
	if chatMap, ok := reputations[chatID]; ok {
		if rep, ok := chatMap[userID]; ok {
			rep.mu.Lock()
			defer rep.mu.Unlock()
			hours := time.Since(rep.lastActivity).Hours()
			if hours > 0 {
				decay := int(hours) * decayAmount
				rep.score += decay
				if rep.score < minScore {
					rep.score = minScore
				}
			}
			rep.lastActivity = time.Now()
			return rep.score
		}
	}
	return initialScore
}

func IncreaseUserScore(chatID int64, userID int, amount int) {
	repMu.Lock()
	defer repMu.Unlock()
	if _, ok := reputations[chatID]; !ok {
		reputations[chatID] = make(map[int]*userReputation)
	}
	rep, ok := reputations[chatID][userID]
	if !ok {
		rep = &userReputation{score: initialScore}
		reputations[chatID][userID] = rep
	}
	rep.mu.Lock()
	defer rep.mu.Unlock()
	rep.score += amount
	if rep.score > maxScore {
		rep.score = maxScore
	}
	rep.lastActivity = time.Now()
}

func DecreaseUserScore(chatID int64, userID int, amount int) {
	IncreaseUserScore(chatID, userID, -amount)
}

func SetUserScoreWithBanHistory(chatID int64, userID int, bannedInOthers bool) {
	score := initialScore
	if bannedInOthers {
		score += banHistoryBonus
	}
	repMu.Lock()
	defer repMu.Unlock()
	if _, ok := reputations[chatID]; !ok {
		reputations[chatID] = make(map[int]*userReputation)
	}
	reputations[chatID][userID] = &userReputation{
		score:        score,
		lastActivity: time.Now(),
	}
}

func IsHighRisk(chatID int64, userID int) bool {
	return GetUserScore(chatID, userID) >= 70
}

func IsLowRisk(chatID int64, userID int) bool {
	return GetUserScore(chatID, userID) < 40
}

func GetLockViolationPenalty(lockType string) int {
	return lockViolation
}

// ---------- پاک‌سازی خودکار کش ----------

// StartReputationCleanup هر ۳۰ دقیقه کاربران غیرفعال (بیش از ۷ روز) را از RAM حذف می‌کند
func StartReputationCleanup() {
	go func() {
		for {
			time.Sleep(30 * time.Minute)
			cleanReputationCache()
		}
	}()
}

func cleanReputationCache() {
	repMu.Lock()
	defer repMu.Unlock()
	now := time.Now()
	maxInactive := 7 * 24 * time.Hour

	for chatID, chatMap := range reputations {
		for userID, rep := range chatMap {
			rep.mu.Lock()
			inactive := now.Sub(rep.lastActivity) > maxInactive
			rep.mu.Unlock()
			if inactive {
				delete(chatMap, userID)
			}
		}
		if len(chatMap) == 0 {
			delete(reputations, chatID)
		}
	}
}
