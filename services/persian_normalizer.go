package services

import (
	"strings"
	"unicode"
)

// حذف اعراب و نشانه‌های حرکت‌گذاری (Mark, Nonspacing)
func removeDiacritics(text string) string {
	var result strings.Builder
	for _, r := range text {
		if !unicode.Is(unicode.Mn, r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// نقشهٔ نرمال‌سازی کاراکترهای خاص
var charMap = map[rune]rune{
	'ك': 'ک', 'ي': 'ی', 'ة': 'ه', 'ؤ': 'و', 'إ': 'ا',
	'أ': 'ا', 'آ': 'ا', 'ئ': 'ی', '0': '۰', '1': '۱',
	'2': '۲', '3': '۳', '4': '۴', '5': '۵', '6': '۶',
	'7': '۷', '8': '۸', '9': '۹',
}

// NormalizeText یک رشته فارسی را برای ورود به مدل استاندارد می‌کند
func NormalizeText(text string) string {
	// حذف اعراب
	text = removeDiacritics(text)

	// یکسان‌سازی کاراکترها
	result := strings.Builder{}
	for _, r := range text {
		if mapped, ok := charMap[r]; ok {
			result.WriteRune(mapped)
		} else {
			result.WriteRune(r)
		}
	}

	// حذف فاصله‌های اضافی
	normalized := strings.TrimSpace(result.String())
	return normalized
}

// Tokenize یک متن نرمال‌شده را به کلمات تشکیل‌دهنده تجزیه می‌کند
func Tokenize(text string) []string {
	text = NormalizeText(text)
	var words []string
	var current strings.Builder

	for _, r := range text {
		if unicode.Is(unicode.Arabic, r) || unicode.IsDigit(r) || r == '_' {
			current.WriteRune(r)
		} else if r == ' ' || r == '\t' || r == '\n' {
			if current.Len() > 0 {
				word := current.String()
				if len([]rune(word)) >= 2 {
					words = append(words, word)
				}
				current.Reset()
			}
		}
	}
	// آخرین کلمه
	if current.Len() > 0 {
		word := current.String()
		if len([]rune(word)) >= 2 {
			words = append(words, word)
		}
	}
	return words
}
