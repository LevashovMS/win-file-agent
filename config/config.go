package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"

	"mediamagi.ru/win-file-agent/log"
)

var cfg atomic.Pointer[cfgData]

type cfgData struct {
	Port        int    `json:"port"`
	WorkerCount int    `json:"worker_count"`
	WorkerQueue int    `json:"worker_queue"`
	TmpDir      string `json:"tmp_dir"`
}

func init() {
	// default
	cfg.Store(&cfgData{
		Port:        8099,
		WorkerCount: 1,
		WorkerQueue: 10,
	})
}

func Load() *cfgData {
	return cfg.Load()
}

func InitFromFile(args ...string) {
	for _, fileName := range []string{"config.json", "config/config.json"} {
		execPath, err := os.Executable()
		if err != nil {
			log.Fatal("Could not find executable path: %+v", err)
		}
		logFilePath := filepath.Join(filepath.Dir(execPath), fileName)
		file, err := os.Open(logFilePath)
		if err != nil {
			//log.Printf("Ошибка открытия файла: %v\n", err)
			continue
		}
		defer file.Close()

		InitFromJson(file)
		return
	}
}

func InitFromJson(data io.Reader) {
	decoder := json.NewDecoder(data)
	var _cfg = new(cfgData)
	if err := decoder.Decode(_cfg); err != nil {
		log.Error("Ошибка декодирования: %+v\n", err)
		return
	}

	cfg.Store(_cfg)
	log.Debug("Загружен конфиг: %+v\n", *_cfg)
}
