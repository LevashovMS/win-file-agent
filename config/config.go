package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync/atomic"

	"log"
)

var Cfg atomic.Pointer[cfg]

type cfg struct {
	Port        int `json:"port"`
	WorkerCount int `json:"worker_count"`
	WorkerQueue int `json:"worker_queue"`
	Ftp         *ftp
}

type ftp struct {
	TmpDir string `json:"tmp_dir"`
	Addr   string `json:"addr"`
	Login  string `json:"login"`
	Pass   string `json:"pass"`
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
		var _cfg = new(cfg)
		if err := decoder.Decode(_cfg); err != nil {
			log.Printf("Ошибка декодирования: %v\n", err)
			return
		}

		Cfg.Store(_cfg)
		log.Printf("Загружен конфиг: %+v\n", *_cfg)
		return
	}
}
