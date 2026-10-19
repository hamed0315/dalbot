package services

import (
	"strings"
)

// RiskInfo نتیجهٔ ارزیابی خطر یک پیام را مشخص می‌کند
type RiskInfo struct {
	IsRisky bool
	Reason  string
	Score   int // 1-10
}

// EvaluateRisk محتوای پیام را برای تشخیص محتوای خطرناک (فحش، +۱۸) بررسی می‌کند
func EvaluateRisk(text string) RiskInfo {
	lowerText := strings.ToLower(text)

	// --------------- ۱. لیست سیاه الفاظ رکیک ---------------
	blacklist := []string{
		// این کلمات صرفاً جهت تست هستند و باید با یک لیست جامع‌تر جایگزین شوند.
		// در نسخهٔ نهایی، این لیست از دیتابیس خوانده خواهد شد.
	}
	for _, word := range blacklist {
		if strings.Contains(lowerText, word) {
			return RiskInfo{IsRisky: true, Reason: "استفاده از الفاظ نامناسب", Score: 5}
		}
	}

	// --------------- ۲. کلیدواژه‌های محتوای +۱۸ ---------------
	adultKeywords := []string{
		"سکس", "سکسی", "پورن", "لخت", "لختی", "شهوتی",
	}
	for _, kw := range adultKeywords {
		if strings.Contains(lowerText, kw) {
			return RiskInfo{IsRisky: true, Reason: "محتوای نامناسب (اخلاقی)", Score: 8}
		}
	}

	// --------------- ۳. استفاده از مدل Bayes در صورت نیاز ---------------
	// (در این نسخه غیرفعال است تا از حساسیت بیش از حد جلوگیری شود)
	// if PredictSpam(text) > 0.9 {
	//     return RiskInfo{IsRisky: true, Reason: "محتوای مشکوک (تشخیص هوش مصنوعی)", Score: 3}
	// }

	return RiskInfo{IsRisky: false, Reason: "", Score: 0}
}
