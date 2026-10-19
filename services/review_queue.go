package services

import (
	"fmt"
	"time"

	"dalbot/api"
	"dalbot/database"
	"dalbot/logger"
)

// AddToReviewQueue adds a suspicious text to the review queue
func AddToReviewQueue(text string, chatID int64, userID int) {
	_, err := database.DB.Exec(
		"INSERT INTO review_queue (text, chat_id, user_id) VALUES (?, ?, ?)",
		text, chatID, userID,
	)
	if err != nil {
		logger.Error.Printf("review_queue insert error: %v", err)
	}
}

// GetReviewQueueCount returns the number of pending reviews
func GetReviewQueueCount() int {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM review_queue WHERE status = 'pending'").Scan(&count)
	return count
}

// SendReviewReportIfNeeded sends a message to the owner if there are enough pending samples
func SendReviewReportIfNeeded(ownerID string) {
	count := GetReviewQueueCount()
	if count >= 50 {
		msg := fmt.Sprintf("📊 %d پیام مشکوک جدید برای یادگیری آماده است.", count)
		keyboard := api.InlineKeyboardMarkup{
			InlineKeyboard: [][]api.InlineKeyboardButton{
				{
					{Text: "👁️ مشاهده و تأیید", CallbackData: "review_queue_view"},
				},
			},
		}
		api.SendMessageWithKeyboard(ownerID, msg, keyboard)
	}
}

// GetFirstPendingReviews returns up to 10 pending reviews
func GetFirstPendingReviews(limit int) []struct {
	ID     int
	Text   string
	ChatID int64
	UserID int
} {
	rows, err := database.DB.Query(
		"SELECT id, text, chat_id, user_id FROM review_queue WHERE status = 'pending' ORDER BY id LIMIT ?",
		limit,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var reviews []struct {
		ID     int
		Text   string
		ChatID int64
		UserID int
	}
	for rows.Next() {
		var r struct {
			ID     int
			Text   string
			ChatID int64
			UserID int
		}
		if err := rows.Scan(&r.ID, &r.Text, &r.ChatID, &r.UserID); err == nil {
			reviews = append(reviews, r)
		}
	}
	return reviews
}

// ApproveReview marks the review as approved and adds the text to spam_corpus
func ApproveReview(id int, text string) {
	database.DB.Exec("UPDATE review_queue SET status = 'approved' WHERE id = ?", id)
	database.DB.Exec("INSERT INTO spam_corpus (text, label) VALUES (?, 'spam')", text)
}

// RejectReview marks the review as rejected and adds the text as ham
func RejectReview(id int, text string) {
	database.DB.Exec("UPDATE review_queue SET status = 'rejected' WHERE id = ?", id)
	database.DB.Exec("INSERT INTO spam_corpus (text, label) VALUES (?, 'ham')", text)
}

// StartReviewQueueMonitor runs periodically to send reports
func StartReviewQueueMonitor(ownerID string) {
	go func() {
		for {
			time.Sleep(1 * time.Hour) // check every hour
			SendReviewReportIfNeeded(ownerID)
		}
	}()
}
