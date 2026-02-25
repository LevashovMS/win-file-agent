package log

import (
	"log"
)

func Debug(format string, v ...any) {
	log.Printf("[D] "+format, v...)
}

func Info(format string, v ...any) {
	log.Printf("[I] "+format, v...)
}

func Error(format string, v ...any) {
	log.Printf("[E] "+format, v...)
}

func Fatal(format string, v ...any) {
	log.Fatalf("[F] "+format, v...)
}

func Panic(format string, v ...any) {
	log.Panicf("[P] "+format, v...)
}
