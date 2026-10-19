package models

import (
	"database/sql"
	"dalbot/database"
)

type Group struct {
	ID        int64
	ChatID    int64
	Title     string
	OwnerID   int64
	IsActive  bool
	CreatedAt string
	ResetAt   string
}

func GetGroup(chatID int64) (*Group, error) {
	row := database.DB.QueryRow("SELECT id, chat_id, title, owner_id, is_active, created_at, daily_reset_at FROM groups WHERE chat_id = ?", chatID)
	var g Group
	err := row.Scan(&g.ID, &g.ChatID, &g.Title, &g.OwnerID, &g.IsActive, &g.CreatedAt, &g.ResetAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &g, nil
}

func CreateGroup(chatID int64, title string, ownerID int64) error {
	_, err := database.DB.Exec("INSERT INTO groups (chat_id, title, owner_id) VALUES (?, ?, ?)", chatID, title, ownerID)
	return err
}

func SetGroupActive(chatID int64, active bool) error {
	_, err := database.DB.Exec("UPDATE groups SET is_active = ? WHERE chat_id = ?", active, chatID)
	return err
}

func SetGroupOwner(chatID int64, newOwnerID int64) error {
	_, err := database.DB.Exec("UPDATE groups SET owner_id = ? WHERE chat_id = ?", newOwnerID, chatID)
	return err
}
