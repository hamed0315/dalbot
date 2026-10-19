package database

func createReviewQueueTable() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS review_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			chat_id INTEGER DEFAULT 0,
			user_id INTEGER DEFAULT 0,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			status TEXT DEFAULT "pending"
		);
	`)
	if err != nil {
		panic("Failed to create review_queue table: " + err.Error())
	}
}
