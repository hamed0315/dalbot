package models

import "dalbot/database"

type AdminLevel string

const (
	LevelOwner       AdminLevel = "owner"
	LevelSeniorAdmin AdminLevel = "senior_admin"
	LevelAdmin       AdminLevel = "admin"
	LevelSpecial     AdminLevel = "special"
)

func AddAdmin(chatID int64, userID int64, level AdminLevel, addedBy int64) error {
	_, err := database.DB.Exec(
		"INSERT OR REPLACE INTO group_admins (chat_id, user_id, level, added_by) VALUES (?, ?, ?, ?)",
		chatID, userID, string(level), addedBy,
	)
	return err
}

func RemoveAdmin(chatID int64, userID int64) error {
	_, err := database.DB.Exec(
		"DELETE FROM group_admins WHERE chat_id = ? AND user_id = ?", chatID, userID,
	)
	return err
}

func GetAdminLevel(chatID int64, userID int64) (AdminLevel, error) {
	row := database.DB.QueryRow(
		"SELECT level FROM group_admins WHERE chat_id = ? AND user_id = ?", chatID, userID,
	)
	var level string
	err := row.Scan(&level)
	if err != nil {
		return "", err
	}
	return AdminLevel(level), nil
}
