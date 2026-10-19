package database

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(dbPath string) {
	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_busy_timeout=10000&_journal_mode=WAL&_cache_size=-8000")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	DB.Exec("PRAGMA synchronous=NORMAL;")
	createTables()
	migrateAntispamSettings()
}

func createTables() {
	tables := `
	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY,
		chat_id INTEGER UNIQUE NOT NULL,
		title TEXT DEFAULT '',
		owner_id INTEGER NOT NULL,
		is_active BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		daily_reset_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS group_admins (
		id INTEGER PRIMARY KEY,
		chat_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		level TEXT NOT NULL DEFAULT 'admin',
		added_by INTEGER NOT NULL,
		FOREIGN KEY(chat_id) REFERENCES groups(chat_id)
	);

	CREATE TABLE IF NOT EXISTS warnings (
		id INTEGER PRIMARY KEY,
		chat_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		target_first_name TEXT DEFAULT '',
		warned_by INTEGER NOT NULL,
		by_first_name TEXT DEFAULT '',
		reason TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS locks (
		id INTEGER PRIMARY KEY,
		chat_id INTEGER NOT NULL,
		lock_type TEXT NOT NULL,
		is_locked BOOLEAN DEFAULT 1,
		FOREIGN KEY(chat_id) REFERENCES groups(chat_id)
	);

	CREATE TABLE IF NOT EXISTS user_bios (
		id INTEGER PRIMARY KEY,
		chat_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		bio_text TEXT NOT NULL,
		added_by INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS learning_data (
		id INTEGER PRIMARY KEY,
		chat_id INTEGER NOT NULL,
		text TEXT NOT NULL,
		is_bad BOOLEAN NOT NULL,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS bot_messages (
		id INTEGER PRIMARY KEY,
		chat_id INTEGER NOT NULL,
		message_id INTEGER NOT NULL,
		delete_at DATETIME NOT NULL,
		FOREIGN KEY(chat_id) REFERENCES groups(chat_id)
	);

	CREATE TABLE IF NOT EXISTS private_users (
		user_id INTEGER PRIMARY KEY,
		first_name TEXT DEFAULT '',
		username TEXT DEFAULT '',
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS mutes (
		chat_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		unmute_at DATETIME NOT NULL,
		PRIMARY KEY (chat_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS bio_likes (
		chat_id INTEGER NOT NULL,
		target_user_id INTEGER NOT NULL,
		liker_user_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (chat_id, target_user_id, liker_user_id)
	);

	CREATE TABLE IF NOT EXISTS antispam_settings (
		chat_id INTEGER PRIMARY KEY,
		flood_max_msgs INTEGER DEFAULT 5,
		flood_seconds INTEGER DEFAULT 5,
		duplicate_max INTEGER DEFAULT 3,
		duplicate_window_seconds INTEGER DEFAULT 5,
		min_spam_length INTEGER DEFAULT 50,
		punishment TEXT DEFAULT 'mute',
		mute_duration_hours INTEGER DEFAULT 99,
		ai_enabled BOOLEAN DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS group_rules (
		chat_id INTEGER PRIMARY KEY,
		rule_text TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS bad_words (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		word TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS user_messages (
		chat_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		message_count INTEGER DEFAULT 1,
		PRIMARY KEY (chat_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS ban_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		target_first_name TEXT DEFAULT '',
		admin_id INTEGER NOT NULL,
		admin_first_name TEXT DEFAULT '',
		reason TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS mute_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		target_first_name TEXT DEFAULT '',
		admin_id INTEGER NOT NULL,
		admin_first_name TEXT DEFAULT '',
		duration_minutes INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS group_forced_channels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		channel_username TEXT NOT NULL,
		channel_title TEXT DEFAULT '',
		join_link TEXT DEFAULT '',
		added_by INTEGER NOT NULL,
		FOREIGN KEY(chat_id) REFERENCES groups(chat_id)
	);

	CREATE TABLE IF NOT EXISTS group_users (
		chat_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		username TEXT DEFAULT '',
		first_name TEXT DEFAULT '',
		last_active DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (chat_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS global_bans (
		user_id INTEGER PRIMARY KEY,
		first_name TEXT DEFAULT '',
		group_count INTEGER DEFAULT 1,
		reason TEXT DEFAULT '',
		banned_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS invite_links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT UNIQUE NOT NULL,
		chat_id INTEGER NOT NULL,
		owner_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS approved_users (
		chat_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		approved_by INTEGER NOT NULL,
		approved_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (chat_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS purge_queue (
		chat_id INTEGER NOT NULL,
		message_id INTEGER NOT NULL,
		created_at TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS spam_corpus (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER DEFAULT 0,
		text TEXT NOT NULL,
		label TEXT NOT NULL DEFAULT 'spam',
		learned_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	createReviewQueueTable()
	_, err := DB.Exec(tables)
	if err != nil {
		log.Fatal("Failed to create tables:", err)
	}
}

func migrateAntispamSettings() {
	alterStatements := []string{
		"ALTER TABLE antispam_settings ADD COLUMN bayes_enabled BOOLEAN DEFAULT 1",
		"ALTER TABLE antispam_settings ADD COLUMN neural_enabled BOOLEAN DEFAULT 1",
		"ALTER TABLE antispam_settings ADD COLUMN multiphase_enabled BOOLEAN DEFAULT 1",
		"ALTER TABLE antispam_settings ADD COLUMN flood_enabled BOOLEAN DEFAULT 1",
		"ALTER TABLE antispam_settings ADD COLUMN duplicate_enabled BOOLEAN DEFAULT 1",
		"ALTER TABLE antispam_settings ADD COLUMN bayes_threshold REAL DEFAULT 0.85",
		"ALTER TABLE antispam_settings ADD COLUMN neural_threshold REAL DEFAULT 0.9",
		"ALTER TABLE antispam_settings ADD COLUMN multiphase_threshold INTEGER DEFAULT 4",
	}
	for _, stmt := range alterStatements {
		_, err := DB.Exec(stmt)
		if err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				log.Println("Migration note:", err)
			}
		}
	}
}
