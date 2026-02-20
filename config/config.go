package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"log"
)

var Config cfg

type cfg struct {
	Port        int `json:"port"`
	WorkerCount int `json:"worker_count"`
	WorkerQueue int `json:"worker_queue"`
}

func Init() {
	for _, fileName := range []string{"config.json", "config/config.json"} {
		execPath, err := os.Executable()
		if err != nil {
			log.Fatal("Could not find executable path:", err)
		}
		logFilePath := filepath.Join(filepath.Dir(execPath), fileName)
		file, err := os.Open(logFilePath)
		if err != nil {
			//log.Printf("Ошибка открытия файла: %v\n", err)
			continue
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&Config); err != nil {
			log.Printf("Ошибка декодирования: %v\n", err)
			return
		}

		log.Printf("Загружен конфиг: %+v\n", Config)
		return
	}
}
