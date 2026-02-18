package controllers

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"log"

	"mediamagi.ru/win-file-agent/worker"
)

type TaskReq struct {
	InDir  string
	OutDir string
	Files  []string
	Cmd    string
	Args   []string
}

func (c *TaskReq) To() *worker.Task {
	return &worker.Task{
		ID:     c.GetID(),
		InDir:  c.InDir,
		OutDir: c.OutDir,
		Urls:   c.Files,
		Cmd:    c.Cmd,
		Args:   c.Args,
	}
}

// Если делать hash то можно отслеживать, что несколько раз кидают одинаковые команды
// если команда уже в работе, то выдавать ошибку.
func (c *TaskReq) GetID() string {
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
