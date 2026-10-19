package logger

import (
	"log"
	"os"
)

var (
	Info    *log.Logger
	Error   *log.Logger
	Monitor *log.Logger
	Debug   *log.Logger
)

func Init() {
	Info    = log.New(os.Stdout, "INFO: ", log.LstdFlags)
	Error   = log.New(os.Stderr, "ERROR: ", log.LstdFlags|log.Lshortfile)
	Monitor = log.New(os.Stdout, "MONITOR: ", log.LstdFlags)
	Debug   = log.New(os.Stdout, "DEBUG: ", log.LstdFlags|log.Lshortfile)
}
