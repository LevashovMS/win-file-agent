//go:build windows
// +build windows

package log

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
)

func Init(ctx context.Context) {
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal("Could not find executable path:", err)
	}
	logFilePath := filepath.Join(filepath.Dir(execPath), "app.log")
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	mw := io.MultiWriter(file) // Writes to both stdout and file
	log.SetOutput(mw)

	// Ждем завершения контекста
	go func() {
		<-ctx.Done()
		file.Close()
	}()
}
