package services

import "sync"

var (
	DefenseActive = make(map[int64]bool)
	DefensePrev   = make(map[int64]PreviousSettings)
	DefenseMu     sync.Mutex
)
