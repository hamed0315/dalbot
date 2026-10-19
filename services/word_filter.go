package services

import (
	"regexp"
	"strings"
	"sync"

	"dalbot/database"
	"dalbot/logger"
)

type badWordEntry struct {
	word     string
	compiled *regexp.Regexp // غیر nil اگر الگوی regex باشد
}

var (
	badWordsCache = make(map[int64][]badWordEntry)
	bwCacheMu     sync.Mutex
)

func InvalidateBadWordsCache(chatID int64) {
	bwCacheMu.Lock()
	delete(badWordsCache, chatID)
	bwCacheMu.Unlock()
}

// ContainsBadWord بررسی می‌کند که آیا متن حاوی کلمهٔ ممنوع (ساده یا regex) هست یا خیر
func ContainsBadWord(chatID int64, text string) bool {
	bwCacheMu.Lock()
	entries, ok := badWordsCache[chatID]
	bwCacheMu.Unlock()

	if !ok {
		entries = loadBadWordsFromDB(chatID)
		bwCacheMu.Lock()
		badWordsCache[chatID] = entries
		bwCacheMu.Unlock()
	}

	lowerText := strings.ToLower(text)
	for _, e := range entries {
		if e.compiled != nil {
			if e.compiled.MatchString(lowerText) {
				return true
			}
		} else {
			if strings.Contains(lowerText, strings.ToLower(e.word)) {
				return true
			}
		}
	}
	return false
}

// loadBadWordsFromDB همهٔ کلمات را از دیتابیس خوانده و regexها را کامپایل می‌کند
func loadBadWordsFromDB(chatID int64) []badWordEntry {
	rows, err := database.DB.Query("SELECT word FROM bad_words WHERE chat_id = ?", chatID)
	if err != nil {
		logger.Error.Printf("loadBadWordsFromDB: %v", err)
		return nil
	}
	defer rows.Close()

	var entries []badWordEntry
	for rows.Next() {
		var word string
		if err := rows.Scan(&word); err != nil {
			continue
		}
		entry := badWordEntry{word: word}
		lower := strings.ToLower(word)
		if strings.HasPrefix(lower, "regex:") {
			pattern := word[6:] // حذف پیشوند "regex:"
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				logger.Error.Printf("regex compile error for '%s': %v", pattern, err)
				// در صورت خطا، همان متن ساده در نظر گرفته می‌شود
				entry.compiled = nil
				entry.word = pattern
			} else {
				entry.compiled = compiled
				entry.word = pattern
			}
		}
		entries = append(entries, entry)
	}
	return entries
}

// AddBadWord کلمهٔ جدید را ذخیره می‌کند (در صورت regex بودن، با پیشوند regex: ذخیره می‌شود)
func AddBadWord(chatID int64, word string) error {
	_, err := database.DB.Exec("INSERT INTO bad_words (chat_id, word) VALUES (?, ?)", chatID, word)
	if err != nil {
		return err
	}
	InvalidateBadWordsCache(chatID)
	return nil
}

// RemoveBadWord کلمه را حذف می‌کند
func RemoveBadWord(chatID int64, word string) error {
	_, err := database.DB.Exec("DELETE FROM bad_words WHERE chat_id = ? AND word = ?", chatID, word)
	if err != nil {
		return err
	}
	InvalidateBadWordsCache(chatID)
	return nil
}
