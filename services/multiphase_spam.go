package services

import (
	"strings"
	"sync"
	"time"
)

// پیام اخیر برای تحلیل چندفازی
type recentMsg struct {
	Text      string
	Timestamp time.Time
}

// کش تاریخچهٔ کوتاه‌مدت
var (
	msgHistory   = make(map[int64]map[int][]recentMsg) // chatID -> userID -> messages
	historyMutex sync.Mutex
)

const (
	maxHistorySize       = 5               // تعداد پیام‌های ذخیره‌شده
	historyWindow        = 30 * time.Second // حداکثر فاصلهٔ زمانی بین پیام‌ها
	multiPhaseThreshold  = 4               // آستانهٔ امتیاز اسپم
)

// CheckMultiPhase analyzes recent messages to detect distributed spam.
func CheckMultiPhase(chatID int64, userID int, text string) bool {
	historyMutex.Lock()
	defer historyMutex.Unlock()

	// اطمینان از وجود map برای این چت
	if _, ok := msgHistory[chatID]; !ok {
		msgHistory[chatID] = make(map[int][]recentMsg)
	}

	// اضافه کردن پیام جدید
	msgHistory[chatID][userID] = append(msgHistory[chatID][userID], recentMsg{
		Text:      text,
		Timestamp: time.Now(),
	})

	// نگه‌داشتن فقط ۵ پیام آخر
	if len(msgHistory[chatID][userID]) > maxHistorySize {
		msgHistory[chatID][userID] = msgHistory[chatID][userID][len(msgHistory[chatID][userID])-maxHistorySize:]
	}

	// اگر کمتر از ۵ پیام داریم، نمی‌توانیم تحلیل کنیم
	if len(msgHistory[chatID][userID]) < maxHistorySize {
		return false
	}

	// بررسی پنجرهٔ زمانی: همهٔ پیام‌ها باید در ۳۰ ثانیه اخیر باشند
	now := time.Now()
	valid := true
	for _, m := range msgHistory[chatID][userID] {
		if now.Sub(m.Timestamp) > historyWindow {
			valid = false
			break
		}
	}
	if !valid {
		return false
	}

	// محاسبهٔ امتیاز اسپم
	score := 0

	// کلمات کلیدی تبلیغاتی
	adKeywords := []string{"کانال", "گروه", "عضو", "لینک", "ربات", "رایگان", "کسب درآمد", "فالوور", "تخفیف", "خرید", "فروش", "ثبت نام"}
	for _, m := range msgHistory[chatID][userID] {
		lower := strings.ToLower(m.Text)
		// لینک
		if strings.Contains(lower, "http") || strings.Contains(lower, "ble.ir") || strings.Contains(lower, "t.me/") {
			score += 3
		}
		// کلمات کلیدی
		for _, kw := range adKeywords {
			if strings.Contains(lower, kw) {
				score += 2
			}
		}
		// عدد (احتمالاً قیمت یا تاریخ)
		for _, ch := range m.Text {
			if ch >= '0' && ch <= '9' {
				score++
				break
			}
		}
	}

	// میانگین طول پیام‌ها
	totalLen := 0
	for _, m := range msgHistory[chatID][userID] {
		totalLen += len([]rune(m.Text))
	}
	avgLen := totalLen / len(msgHistory[chatID][userID])
	if avgLen < 20 {
		score += 2 // پیام‌های خیلی کوتاه مشکوک‌ترند
	}

	return score >= multiPhaseThreshold
}

// GetUserRecentMessages returns the recent messages for a user (used for deletion).
func GetUserRecentMessages(chatID int64, userID int) []string {
	historyMutex.Lock()
	defer historyMutex.Unlock()

	if chatMap, ok := msgHistory[chatID]; ok {
		if msgs, ok := chatMap[userID]; ok {
			result := make([]string, len(msgs))
			for i, m := range msgs {
				result[i] = m.Text
			}
			return result
		}
	}
	return nil
}

// CleanupOldHistory removes entries older than 60 seconds and empty chats/users.
func CleanupOldHistory() {
	historyMutex.Lock()
	defer historyMutex.Unlock()

	cutoff := time.Now().Add(-60 * time.Second)
	for chatID, chatMap := range msgHistory {
		for userID, msgs := range chatMap {
			var fresh []recentMsg
			for _, m := range msgs {
				if m.Timestamp.After(cutoff) {
					fresh = append(fresh, m)
				}
			}
			if len(fresh) == 0 {
				delete(chatMap, userID)
			} else {
				chatMap[userID] = fresh
			}
		}
		if len(chatMap) == 0 {
			delete(msgHistory, chatID)
		}
	}
}

// StartMultiPhaseCleanupWorker runs periodic cleanup.
func StartMultiPhaseCleanupWorker() {
	ticker := time.NewTicker(2 * time.Minute)
	go func() {
		for range ticker.C {
			CleanupOldHistory()
		}
	}()
}
