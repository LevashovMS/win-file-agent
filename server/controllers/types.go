package controllers

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"mediamagi.ru/win-file-agent/worker"
)

const oneGB uint64 = 1000 * 1000 * 1000

type TaskReq struct {
	InDir  string   `json:"in_dir"`
	OutDir string   `json:"out_dir"`
	Urls   []string `json:"urls"`
	Cmd    string   `json:"cmd"`
	Args   []string `json:"args"`
	OutExt string   `json:"out_ext"`
}

func (c *TaskReq) To() *worker.Task {
	return &worker.Task{
		ID:     c.getID(),
		InDir:  c.InDir,
		OutDir: c.OutDir,
		Urls:   c.Urls,
		Cmd:    c.Cmd,
		Args:   c.Args,
		OutExt: c.OutExt,
	}
}

func (c *TaskReq) Verification() error {
	var msg []string
	if len(c.InDir) == 0 {
		msg = append(msg, "Не задана входящая папка")
	}
	if len(c.OutDir) == 0 {
		msg = append(msg, "Не задана исходящая папка")
	}
	if len(c.Urls) == 0 {
		msg = append(msg, "Не задан(ы) файлы для скачивания")
	}
	for _, rawURL := range c.Urls {
		u, err := url.ParseRequestURI(rawURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			msg = append(msg, fmt.Sprintf("Некорректный URL: %s", rawURL))
		}
	}
	if len(c.Cmd) == 0 {
		msg = append(msg, "Не задана команда запуска")
	}
	// TODO точно надо проверять?
	if len(c.Args) == 0 {
		msg = append(msg, "Не задан(ы) аргументы для команды")
	}

	if len(msg) > 0 {
		return errors.New(strings.Join(msg, " "))
	}

	if len(c.OutExt) > 0 && c.OutExt[0] != '.' {
		c.OutExt = "." + c.OutExt
	}

	return nil
}

// Если делать hash то можно отслеживать, что несколько раз кидают одинаковые команды
// если команда уже в работе, то выдавать ошибку.
func (c *TaskReq) getID() string {
	//return fmt.Sprintf("%d", time.Now().Unix())

	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(c); err != nil {
		log.Printf("Обшибка создания ID, %+v", err)
		return ""
	}

	var hash = sha256.New()
	hash.Write(b.Bytes())
	return hex.EncodeToString(hash.Sum(nil))
}
