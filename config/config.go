package config

import (
	"encoding/json"
	"os"

	"log"
)

var Config cfg

type cfg struct {
	Port        int `json:"port"`
	WorkerCount int `json:"worker_count"`
}

func init() {
	for _, fileName := range []string{"config.json", "config/config.json"} {
		file, err := os.Open(fileName)
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
