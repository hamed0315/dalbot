package services

import (
	"math"
	"strings"
	"sync"
	"time"
	"unicode"

	"dalbot/database"
	"dalbot/logger"
)

type spamModel struct {
	mu        sync.RWMutex
	spamWords map[string]int
	hamWords  map[string]int
	totalSpam int
	totalHam  int
	trained   bool
}

var learner = &spamModel{
	spamWords: make(map[string]int),
	hamWords:  make(map[string]int),
}

func InitLearner() {
	loadAndTrain()
	// اگر پایگاه داده خالی بود، داده‌های اولیه را اضافه کن
	learner.mu.RLock()
	empty := (learner.totalSpam == 0 && learner.totalHam == 0)
	learner.mu.RUnlock()
	if empty {
		seedInitialData()
		loadAndTrain()
	}
	go periodicRetrain()
}

func seedInitialData() {
	spamSamples := []string{
		"کانال ما عضو شوید",
		"کسب درآمد میلیونی در منزل",
		"ربات رایگان تلگرام",
		"لینک گروه سکسی",
		"فروش فالوور ارزان",
		"همین الان عضو شو",
		"جوایز نقدی",
		"برنامه حسابداری رایگان",
		"پکیج آموزش کسب و کار",
		"آی پی ثابت",
	}
	hamSamples := []string{
		"سلام خوبی",
		"چه خبر",
		"مرسی",
		"امروز چطور بود",
		"خدا رو شکر",
		"فردا میایم",
		"آره حتما",
		"قربانت",
		"شب بخیر",
		"صبح بخیر",
	}

	for _, s := range spamSamples {
		database.DB.Exec("INSERT INTO spam_corpus (chat_id, text, label) VALUES (0, ?, 'spam')", s)
	}
	for _, s := range hamSamples {
		database.DB.Exec("INSERT INTO spam_corpus (chat_id, text, label) VALUES (0, ?, 'ham')", s)
	}
	logger.Info.Println("spam_learner: seed data inserted")
}

func LearnSpam(text string) {
	learner.mu.Lock()
	defer learner.mu.Unlock()

	words := tokenize(text)
	for _, w := range words {
		learner.spamWords[w]++
	}
	learner.totalSpam++
	learner.trained = false
}

func LearnHam(text string) {
	learner.mu.Lock()
	defer learner.mu.Unlock()

	words := tokenize(text)
	for _, w := range words {
		learner.hamWords[w]++
	}
	learner.totalHam++
	learner.trained = false
}

func PredictSpam(text string) float64 {
	learner.mu.RLock()
	defer learner.mu.RUnlock()

	if !learner.trained || (learner.totalSpam == 0 && learner.totalHam == 0) {
		return 0.0
	}

	words := tokenize(text)
	spamProb := math.Log(float64(learner.totalSpam+1) / float64(learner.totalSpam+learner.totalHam+2))
	hamProb := math.Log(float64(learner.totalHam+1) / float64(learner.totalSpam+learner.totalHam+2))

	for _, w := range words {
		spamCount := learner.spamWords[w]
		hamCount := learner.hamWords[w]
		pWordSpam := float64(spamCount+1) / float64(learner.totalSpam+2)
		pWordHam := float64(hamCount+1) / float64(learner.totalHam+2)
		spamProb += math.Log(pWordSpam)
		hamProb += math.Log(pWordHam)
	}

	sum := spamProb + hamProb
	if sum == 0 {
		return 0.5
	}
	prob := math.Exp(spamProb) / (math.Exp(spamProb) + math.Exp(hamProb))
	return prob
}

func loadAndTrain() {
	learner.mu.Lock()
	defer learner.mu.Unlock()

	learner.spamWords = make(map[string]int)
	learner.hamWords = make(map[string]int)
	learner.totalSpam = 0
	learner.totalHam = 0

	rows, err := database.DB.Query("SELECT text, label FROM spam_corpus")
	if err != nil {
		logger.Error.Printf("spam_learner: load corpus error: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var text, label string
		if err := rows.Scan(&text, &label); err != nil {
			continue
		}
		words := tokenize(text)
		if label == "spam" {
			for _, w := range words {
				learner.spamWords[w]++
			}
			learner.totalSpam++
		} else if label == "ham" {
			for _, w := range words {
				learner.hamWords[w]++
			}
			learner.totalHam++
		}
	}
	learner.trained = true
	logger.Info.Printf("spam_learner: trained with %d spam, %d ham", learner.totalSpam, learner.totalHam)
}

func periodicRetrain() {
	for {
		time.Sleep(10 * time.Minute)
		loadAndTrain()
	}
}

func tokenize(text string) []string {
	var words []string
	text = strings.ToLower(text)
	word := ""
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			word += string(r)
		} else {
			if len(word) > 1 && len(word) < 30 {
				words = append(words, word)
			}
			word = ""
		}
	}
	if len(word) > 1 && len(word) < 30 {
		words = append(words, word)
	}
	return words
}
