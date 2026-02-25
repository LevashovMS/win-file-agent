package controllers

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"mediamagi.ru/win-file-agent/config"
	"mediamagi.ru/win-file-agent/errors"
	"mediamagi.ru/win-file-agent/log"
	"mediamagi.ru/win-file-agent/worker"
)

const oneGB uint64 = 1000 * 1000 * 1000

type TaskReq struct {
	InDir  string      `json:"in_dir"`
	OutDir string      `json:"out_dir"`
	Urls   []string    `json:"urls"`
	Cmd    string      `json:"cmd"`
	Args   []string    `json:"args"`
	OutExt string      `json:"out_ext"`
	Ftp    *worker.Ftp `json:"ftp"`

	isSaveToFtp bool `json:"-"`
}

func (c *TaskReq) ToWTask() *worker.Task {
	var t = &worker.Task{
		ID:     c.getID(),
		InDir:  c.InDir,
		OutDir: c.OutDir,
		Urls:   c.Urls,
		Cmd:    c.Cmd,
		Args:   c.Args,
		OutExt: c.OutExt,
	}
	if c.isSaveToFtp {
		t.SaveToFtp(c.Ftp)
	}

	return t
}

func (c *TaskReq) verification() error {
	var msg []string
	if len(c.InDir) == 0 {
		msg = append(msg, "Не задана входящая папка")
	}
	if len(c.OutDir) == 0 {
		if c.Ftp == nil {
			msg = append(msg, "Не задана исходящая папка")
			msg = append(msg, "Не заданы настройки ftp")
		}
		if len(config.Cfg.Load().TmpDir) == 0 {
			msg = append(msg, "Не задано в настройках сервиса временное хранение файлов")
		}
		if len(c.Ftp.Addr) == 0 {
			msg = append(msg, "Не задан адрес ftp сервера")
		}
		c.isSaveToFtp = true
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
		log.Error("Обшибка создания ID, %+v", errors.WithStack(err))
		return ""
	}

	var hash = sha256.New()
	hash.Write(b.Bytes())
	return hex.EncodeToString(hash.Sum(nil))
}
