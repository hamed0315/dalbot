package models

import "dalbot/database"

func AddWarning(chatID int64, userID int64, targetFirstName string, warnedBy int64, byFirstName string, reason string) error {
	_, err := database.DB.Exec(
		"INSERT INTO warnings (chat_id, user_id, target_first_name, warned_by, by_first_name, reason) VALUES (?, ?, ?, ?, ?, ?)",
		chatID, userID, targetFirstName, warnedBy, byFirstName, reason,
	)
	return err
}

func GetWarningCount(chatID int64, userID int64) (int, error) {
	row := database.DB.QueryRow(
		"SELECT COUNT(*) FROM warnings WHERE chat_id = ? AND user_id = ?", chatID, userID,
	)
	var count int
	err := row.Scan(&count)
	return count, err
}

func ClearWarnings(chatID int64, userID int64) error {
	_, err := database.DB.Exec(
		"DELETE FROM warnings WHERE chat_id = ? AND user_id = ?", chatID, userID,
	)
	return err
}
