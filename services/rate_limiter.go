package services

import (
	"sync/atomic"
	"time"
)

var (
	tokens        int64 = 0
	criticalTokens      = make(chan struct{}, 15) // توکن‌های اضطراری
)

func StartRateLimiter(ratePerSec int) {
	// تولید توکن‌های عادی
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(ratePerSec))
		defer ticker.Stop()
		for range ticker.C {
			atomic.AddInt64(&tokens, 1)
			if atomic.LoadInt64(&tokens) > int64(ratePerSec*2) {
				atomic.StoreInt64(&tokens, int64(ratePerSec*2))
			}
		}
	}()
	// تولید توکن‌های بحرانی (هر ۱۰۰ میلی‌ثانیه یک توکن، تا ۱۵)
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case criticalTokens <- struct{}{}:
			default:
			}
		}
	}()
}

// AcquireToken برای عملیات عادی
func AcquireToken() {
	for atomic.LoadInt64(&tokens) <= 0 {
		time.Sleep(10 * time.Millisecond)
	}
	atomic.AddInt64(&tokens, -1)
}

// AcquireCriticalToken برای عملیات حیاتی (حذف اسپم، بن)
func AcquireCriticalToken() {
	<-criticalTokens
}
