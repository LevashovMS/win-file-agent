package log

import (
	"io"
	"log"
	"os"
)

var file *os.File

func init() {
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	//defer file.Close()

	mw := io.MultiWriter(os.Stdout, file) // Writes to both stdout and file
	log.SetOutput(mw)
}

func FileClose() {
	file.Close()
}
