package services

import (
	"dalbot/database"
	"dalbot/logger"
)

type LockPunishment struct {
	Punishment      string
	MuteDurationHrs int
}

var lockPunishmentCache = make(map[int64]map[string]LockPunishment)

func GetLockPunishment(chatID int64, lockType string) LockPunishment {
	if chatCache, ok := lockPunishmentCache[chatID]; ok {
		if lp, ok := chatCache[lockType]; ok {
			return lp
		}
	}
	var lp LockPunishment
	err := database.DB.QueryRow("SELECT punishment, mute_duration_hours FROM lock_punishments WHERE chat_id = ? AND lock_type = ?", chatID, lockType).Scan(&lp.Punishment, &lp.MuteDurationHrs)
	if err != nil {
		lp = LockPunishment{Punishment: "delete", MuteDurationHrs: 1}
	}
	if lockPunishmentCache[chatID] == nil {
		lockPunishmentCache[chatID] = make(map[string]LockPunishment)
	}
	lockPunishmentCache[chatID][lockType] = lp
	return lp
}

func SetLockPunishment(chatID int64, lockType string, punishment string, muteDuration int) {
	_, err := database.DB.Exec("INSERT OR REPLACE INTO lock_punishments (chat_id, lock_type, punishment, mute_duration_hours) VALUES (?, ?, ?, ?)", chatID, lockType, punishment, muteDuration)
	if err != nil {
		logger.Error.Printf("SetLockPunishment failed: %v", err)
		return
	}
	if lockPunishmentCache[chatID] == nil {
		lockPunishmentCache[chatID] = make(map[string]LockPunishment)
	}
	lockPunishmentCache[chatID][lockType] = LockPunishment{Punishment: punishment, MuteDurationHrs: muteDuration}
}

func ApplyLockPunishment(chatID int64, userID int, firstName string, lockType string) {
	lp := GetLockPunishment(chatID, lockType)
	switch lp.Punishment {
	case "warn":
		// warnUser is in handlers, so we need a bridge. We'll call it from handlers.
		// To avoid circular imports, we'll handle this in handlers.go directly.
	case "mute":
		// Will be handled in handlers.go
	case "ban":
		// Will be handled in handlers.go
	}
}
