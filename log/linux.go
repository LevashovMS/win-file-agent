//go:build !windows
// +build !windows

package log

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

func Init(ctx context.Context, fileName string) {
	sync.OnceFunc(func() {
		execPath, err := os.Executable()
		if err != nil {
			log.Fatal("Could not find executable path:", err)
		}
		if len(fileName) == 0 {
			fileName = "app.log"
		}

		logFilePath := filepath.Join(filepath.Dir(execPath), fileName)
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}

		mw := io.MultiWriter(os.Stdout, file) // Writes to both stdout and file
		log.SetOutput(mw)

		// Ждем завершения контекста
		go func() {
			<-ctx.Done()
			file.Close()
		}()
	})()
}
