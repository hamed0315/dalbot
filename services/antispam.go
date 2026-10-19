package services

import (
	"database/sql"
	"sync"
	"time"

	"dalbot/database"
	"dalbot/logger"
)

type AntispamConfig struct {
	FloodMaxMsgs        int
	FloodSeconds        int
	DuplicateMax        int
	DuplicateWindowSec  int
	MinSpamLength       int
	Punishment          string
	MuteDurationHrs     int
	AiEnabled           bool
	BayesEnabled        bool
	NeuralEnabled       bool
	MultiphaseEnabled   bool
	FloodEnabled        bool
	DuplicateEnabled    bool
	BayesThreshold      float64
	NeuralThreshold     float64
	MultiphaseThreshold int
}

var defaultConfig = AntispamConfig{
	FloodMaxMsgs:        5,
	FloodSeconds:        5,
	DuplicateMax:        4,
	DuplicateWindowSec:  5,
	MinSpamLength:       50,
	Punishment:          "warn",
	MuteDurationHrs:     1,
	AiEnabled:           true,
	BayesEnabled:        true,
	NeuralEnabled:       true,
	MultiphaseEnabled:   true,
	FloodEnabled:        true,
	DuplicateEnabled:    true,
	BayesThreshold:      0.85,
	NeuralThreshold:     0.9,
	MultiphaseThreshold: 4,
}

var configCache = struct {
	sync.Mutex
	entries map[int64]*AntispamConfig
}{entries: make(map[int64]*AntispamConfig)}

func GetAntispamConfig(chatID int64) *AntispamConfig {
	configCache.Lock()
	defer configCache.Unlock()
	if cfg, ok := configCache.entries[chatID]; ok {
		return cfg
	}
	cfg := loadConfigFromDB(chatID)
	configCache.entries[chatID] = cfg
	return cfg
}

func InvalidateConfig(chatID int64) {
	configCache.Lock()
	delete(configCache.entries, chatID)
	configCache.Unlock()
}

func loadConfigFromDB(chatID int64) *AntispamConfig {
	cfg := defaultConfig
	row := database.DB.QueryRow(
		`SELECT flood_max_msgs, flood_seconds, duplicate_max,
			duplicate_window_seconds, min_spam_length,
			punishment, mute_duration_hours, ai_enabled,
			COALESCE(bayes_enabled, 1), COALESCE(neural_enabled, 1),
			COALESCE(multiphase_enabled, 1), COALESCE(flood_enabled, 1),
			COALESCE(duplicate_enabled, 1), COALESCE(bayes_threshold, 0.85),
			COALESCE(neural_threshold, 0.9), COALESCE(multiphase_threshold, 4)
		FROM antispam_settings WHERE chat_id = ?`,
		chatID,
	)
	var fm, fs, dm, dws, msl, mdh int
	var p string
	var ai bool
	var bayesEn, neuralEn, multiEn, floodEn, dupEn bool
	var bayesTh, neuralTh float64
	var multiTh int
	err := row.Scan(&fm, &fs, &dm, &dws, &msl, &p, &mdh, &ai,
		&bayesEn, &neuralEn, &multiEn, &floodEn, &dupEn,
		&bayesTh, &neuralTh, &multiTh)
	if err == nil {
		cfg.FloodMaxMsgs = fm
		cfg.FloodSeconds = fs
		cfg.DuplicateMax = dm
		cfg.DuplicateWindowSec = dws
		cfg.MinSpamLength = msl
		cfg.Punishment = p
		cfg.MuteDurationHrs = mdh
		cfg.AiEnabled = ai
		cfg.BayesEnabled = bayesEn
		cfg.NeuralEnabled = neuralEn
		cfg.MultiphaseEnabled = multiEn
		cfg.FloodEnabled = floodEn
		cfg.DuplicateEnabled = dupEn
		cfg.BayesThreshold = bayesTh
		cfg.NeuralThreshold = neuralTh
		cfg.MultiphaseThreshold = multiTh
	} else if err != sql.ErrNoRows {
		// ignore
	}
	return &cfg
}

// ---------- Flood / Duplicate Tracker ----------
type floodRecord struct {
	mu          sync.Mutex
	messages    []time.Time
	lastText    string
	dupCount    int
	lastDupTime time.Time
}

type floodTracker struct {
	mu    sync.Mutex
	chats map[int64]map[int]*floodRecord
}

var tracker = &floodTracker{
	chats: make(map[int64]map[int]*floodRecord),
}

func init() {
	go tracker.cleanup(30 * time.Second)
}

func (ft *floodTracker) getRecord(chatID int64, userID int) *floodRecord {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	chatMap, exists := ft.chats[chatID]
	if !exists {
		chatMap = make(map[int]*floodRecord)
		ft.chats[chatID] = chatMap
	}

	rec, exists := chatMap[userID]
	if !exists {
		rec = &floodRecord{}
		chatMap[userID] = rec
	}
	return rec
}

func (ft *floodTracker) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		ft.mu.Lock()
		now := time.Now()
		for chatID, chatMap := range ft.chats {
			for userID, rec := range chatMap {
				rec.mu.Lock()
				var fresh []time.Time
				for _, t := range rec.messages {
					if now.Sub(t) < 10*time.Second {
						fresh = append(fresh, t)
					}
				}
				rec.messages = fresh
				if len(fresh) == 0 {
					rec.dupCount = 0
					rec.lastText = ""
				}
				empty := len(fresh) == 0 && rec.dupCount == 0
				rec.mu.Unlock()

				if empty {
					delete(chatMap, userID)
				}
			}
			if len(chatMap) == 0 {
				delete(ft.chats, chatID)
			}
		}
		ft.mu.Unlock()
	}
}

func CheckSpam(chatID int64, userID int, text string) bool {
	cfg := GetAntispamConfig(chatID)

	if cfg.MinSpamLength > 0 {
		if len([]rune(text)) < cfg.MinSpamLength {
			return false
		}
	}

	rec := tracker.getRecord(chatID, userID)
	rec.mu.Lock()
	defer rec.mu.Unlock()

	now := time.Now()

	// --- Flood ---
	var recent []time.Time
	for _, t := range rec.messages {
		if now.Sub(t) <= time.Duration(cfg.FloodSeconds)*time.Second {
			recent = append(recent, t)
		}
	}
	rec.messages = append(recent, now)

	if len(rec.messages) > cfg.FloodMaxMsgs {
		rec.lastText = ""
		rec.dupCount = 0
		return true
	}

	// --- Duplicate ---
	duplicateWindow := time.Duration(cfg.DuplicateWindowSec) * time.Second

	if text == rec.lastText {
		if rec.dupCount > 0 && now.Sub(rec.lastDupTime) <= duplicateWindow {
			rec.dupCount++
		} else {
			rec.dupCount = 1
		}
	} else {
		rec.lastText = text
		rec.dupCount = 1
	}
	rec.lastDupTime = now

	if rec.dupCount > cfg.DuplicateMax {
		return true
	}

	return false
}

// ---------- Presets ----------
type AntispamPreset struct {
	Name                string
	FloodMaxMsgs        int
	FloodSeconds        int
	DuplicateMax        int
	DuplicateWindowSec  int
	BayesThreshold      float64
	NeuralThreshold     float64
	MultiphaseThreshold int
}

var Presets = map[string]AntispamPreset{
	"standard": {
		Name:                "استاندارد 🟢",
		FloodMaxMsgs:        6,
		FloodSeconds:        6,
		DuplicateMax:        4,
		DuplicateWindowSec:  8,
		BayesThreshold:      0.85,
		NeuralThreshold:     0.90,
		MultiphaseThreshold: 4,
	},
	"medium": {
		Name:                "متوسط 🟡",
		FloodMaxMsgs:        5,
		FloodSeconds:        5,
		DuplicateMax:        3,
		DuplicateWindowSec:  6,
		BayesThreshold:      0.80,
		NeuralThreshold:     0.85,
		MultiphaseThreshold: 3,
	},
	"strict": {
		Name:                "سختگیرانه 🔴",
		FloodMaxMsgs:        4,
		FloodSeconds:        4,
		DuplicateMax:        2,
		DuplicateWindowSec:  4,
		BayesThreshold:      0.70,
		NeuralThreshold:     0.75,
		MultiphaseThreshold: 2,
	},
}

func ApplyPreset(chatID int64, preset string) {
	p, ok := Presets[preset]
	if !ok {
		return
	}
	_, err := database.DB.Exec(`
		INSERT OR REPLACE INTO antispam_settings 
		(chat_id, flood_max_msgs, flood_seconds, duplicate_max, duplicate_window_seconds, 
		 bayes_threshold, neural_threshold, multiphase_threshold,
		 flood_enabled, duplicate_enabled, bayes_enabled, neural_enabled, multiphase_enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, 1, 1, 1, 1)`,
		chatID, p.FloodMaxMsgs, p.FloodSeconds, p.DuplicateMax, p.DuplicateWindowSec,
		p.BayesThreshold, p.NeuralThreshold, p.MultiphaseThreshold,
	)
	if err != nil {
		logger.Error.Printf("ApplyPreset error: %v", err)
	}
	InvalidateConfig(chatID)
}
