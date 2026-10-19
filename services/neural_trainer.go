package services

import (
	"time"

	"dalbot/database"
	"dalbot/logger"
)

const (
	weightsFile = "neural_weights.json"
	trainLR     = 0.05    // نرخ یادگیری
	trainEpochs = 5       // تعداد عبور از کل داده
)

// InitNeuralEngine راه‌اندازی مدل عصبی، بارگذاری وزن‌ها و شروع آموزش دوره‌ای
func InitNeuralEngine() {
	globalNN = NewNeuralNet(maxFeatures, 64) // 1000→64→1
	globalVocab = NewVocab()

	// تلاش برای بارگذاری وزن‌های ذخیره‌شده
	if err := globalNN.LoadWeights(weightsFile); err == nil {
		globalNN.Trained = true
		logger.Info.Println("neural: loaded saved weights")
	} else {
		logger.Info.Println("neural: no saved weights, starting fresh")
	}

	// اولین ساخت واژگان و آموزش (اگر داده موجود باشد)
	corpus := loadSpamCorpus()
	if len(corpus) > 0 {
		globalVocab.BuildVocabulary(corpus)
		trainNeural(corpus)
	}

	// آموزش دوره‌ای هر ۲۰ دقیقه
	go func() {
		for {
			time.Sleep(20 * time.Minute)
			corpus := loadSpamCorpus()
			if len(corpus) == 0 {
				continue
			}
			globalVocab.BuildVocabulary(corpus)
			trainNeural(corpus)
		}
	}()
}

// loadSpamCorpus همه متون و برچسب‌ها را از spam_corpus می‌خواند
func loadSpamCorpus() []string {
	rows, err := database.DB.Query("SELECT text FROM spam_corpus")
	if err != nil {
		logger.Error.Printf("neural: load corpus error: %v", err)
		return nil
	}
	defer rows.Close()

	var texts []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err == nil {
			texts = append(texts, text)
		}
	}
	return texts
}

// trainNeural مدل را روی کل پیکره آموزش می‌دهد
func trainNeural(corpus []string) {
	if globalNN == nil || globalVocab == nil || !globalVocab.ready {
		return
	}

	logger.Info.Printf("neural: starting training on %d samples", len(corpus))

	for epoch := 0; epoch < trainEpochs; epoch++ {
		// خواندن مجدد با label
		rows, err := database.DB.Query("SELECT text, label FROM spam_corpus")
		if err != nil {
			return
		}
		count := 0
		for rows.Next() {
			var text, label string
			if err := rows.Scan(&text, &label); err != nil {
				continue
			}
			vec := globalVocab.Vectorize(text)
			target := 0.0
			if label == "spam" {
				target = 1.0
			}
			globalNN.Train(vec, target, trainLR)
			count++
		}
		rows.Close()
		logger.Debug.Printf("neural: epoch %d/%d done (%d samples)", epoch+1, trainEpochs, count)
	}

	globalNN.Trained = true
	if err := globalNN.SaveWeights(weightsFile); err != nil {
		logger.Error.Printf("neural: failed to save weights: %v", err)
	} else {
		logger.Info.Println("neural: training complete, weights saved")
	}
}

// PredictNeural بردار یک متن را گرفته و خروجی شبکه (احتمال اسپم) را برمی‌گرداند
func PredictNeural(text string) (float64, bool) {
	if globalNN == nil || !globalNN.Trained || globalVocab == nil || !globalVocab.ready {
		return 0, false
	}
	vec := globalVocab.Vectorize(text)
	prob := globalNN.Forward(vec)
	return prob, true
}
