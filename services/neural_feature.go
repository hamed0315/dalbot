package services

import (
	"math"
	"sort"
	"sync"
)

const (
	vocabSize   = 1000   // تعداد کلمات پرتکرار
	maxFeatures = vocabSize
)

type Vocab struct {
	mu        sync.RWMutex
	words     []string          // فهرست کلمات (اندیس → کلمه)
	wordToIdx map[string]int    // کلمه → اندیس
	idf       []float64         // IDF هر کلمه
	totalDocs int               // تعداد کل اسناد
	ready     bool
}

var globalVocab *Vocab

func NewVocab() *Vocab {
	return &Vocab{
		wordToIdx: make(map[string]int),
	}
}

// BuildVocabulary از روی تمام متون موجود در spam_corpus واژگان را می‌سازد
func (v *Vocab) BuildVocabulary(corpus []string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// شمارش فراوانی کلمات در کل پیکره
	freq := make(map[string]int)
	docFreq := make(map[string]int) // تعداد اسنادی که کلمه در آن ظاهر شده

	for _, text := range corpus {
		words := Tokenize(text) // از persian_normalizer.go
		seen := make(map[string]bool)
		for _, w := range words {
			freq[w]++
			if !seen[w] {
				docFreq[w]++
				seen[w] = true
			}
		}
	}

	// انتخاب ۱۰۰۰ کلمه برتر بر اساس فراوانی کل
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range freq {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	// محدود به vocabSize
	if len(sorted) > vocabSize {
		sorted = sorted[:vocabSize]
	}

	v.words = make([]string, len(sorted))
	v.wordToIdx = make(map[string]int)
	for i, item := range sorted {
		v.words[i] = item.Key
		v.wordToIdx[item.Key] = i
	}

	// محاسبه IDF
	totalDocs := len(corpus)
	if totalDocs == 0 {
		totalDocs = 1
	}
	v.totalDocs = totalDocs
	v.idf = make([]float64, len(v.words))
	for i, word := range v.words {
		df := docFreq[word]
		if df == 0 {
			df = 1
		}
		v.idf[i] = math.Log(float64(totalDocs) / float64(df))
	}

	v.ready = true
}

// Vectorize یک متن را به بردار TF‑IDF با طول vocabSize تبدیل می‌کند
func (v *Vocab) Vectorize(text string) []float64 {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.ready {
		return make([]float64, maxFeatures)
	}

	words := Tokenize(text)
	// شمارش TF
	tf := make(map[string]int)
	for _, w := range words {
		tf[w]++
	}
	if len(tf) == 0 {
		return make([]float64, maxFeatures)
	}
	totalWords := len(words)

	vec := make([]float64, maxFeatures)
	for word, count := range tf {
		idx, ok := v.wordToIdx[word]
		if !ok {
			continue
		}
		tfVal := float64(count) / float64(totalWords)
		vec[idx] = tfVal * v.idf[idx]
	}
	return vec
}
