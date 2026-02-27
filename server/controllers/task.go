package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"mediamagi.ru/win-file-agent/disk"
	"mediamagi.ru/win-file-agent/errors"
	"mediamagi.ru/win-file-agent/server"
	"mediamagi.ru/win-file-agent/store"
	"mediamagi.ru/win-file-agent/worker"
)

type Task struct {
	w     *worker.Worker
	store store.Store[string, *worker.Task]
}

func NewTask(store store.Store[string, *worker.Task], w *worker.Worker) *Task {
	return &Task{
		store: store,
		w:     w,
	}
}

// Get, "/v1/task" - получение списка ключей всех заданий в работе
func (c *Task) GetAll(req *http.Request) (*[]string, error) {
	var ks = c.store.GetKeys()
	return &ks, nil
}

// Get, "/v1/task/{id}" - получение задание и его статус.
func (c *Task) Get(req *http.Request) (*worker.Task, error) {
	var id = req.PathValue("id")
	if len(id) == 0 {
		return nil, server.StatusCode(http.StatusBadRequest)
	}

	if v, ok := c.store.Load(id); ok {
		return v, nil
	}

	return nil, nil
}

// Post, "/v1/task" - создание задания на обработку.
func (c *Task) Create(req *http.Request) (*string, error) {
	defer req.Body.Close()
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, server.StatusErr(http.StatusBadRequest, err)
	}

	var t = new(TaskReq)
	if err = json.Unmarshal(bodyBytes, t); err != nil {
		return nil, errors.WithStack(err)
	}
	if err = t.verification(); err != nil {
		return nil, server.StatusMsgErr(http.StatusBadRequest, err.Error(), err)
	}

	fs, err := disk.GetFreeSpace(t.InDir)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if fs < oneGB {
		return nil, server.StatusCode(http.StatusInsufficientStorage)
	}

	var tw = t.ToWTask()
	if _, ok := c.store.Load(tw.ID); ok {
		return nil, server.StatusMsgErr(http.StatusConflict, fmt.Sprintf("Задача с таких hash %s в работе.", tw.ID), nil)
	}

	c.w.ExecTask(tw)

	return &tw.ID, server.StatusCode(http.StatusCreated)
}

// Delete, "/v1/task/{id}" отмена задания. {id}
func (c *Task) Delete(req *http.Request) (*any, error) {
	var id = req.PathValue("id")
	if len(id) == 0 {
		return nil, server.StatusCode(http.StatusBadRequest)
	}

	var ok, _ = c.w.StopProc(id)
	if !ok {
		return nil, server.StatusCode(http.StatusNoContent)
	}

	return nil, nil
}
