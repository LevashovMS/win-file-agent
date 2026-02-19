package controllers

import (
	"encoding/json"
	"io"
	"net/http"

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
func (c *Task) GetAll(req *http.Request) (any, error) {
	return c.store.GetKeys(), nil
}

// Get, "/v1/task/{id}" - получение задание и его статус.
func (c *Task) Get(req *http.Request) (any, error) {
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
func (c *Task) Create(req *http.Request) (any, error) {
	defer req.Body.Close()
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		//http.Error(w, "can't read body", http.StatusBadRequest)
		return nil, server.StatusCode(http.StatusBadRequest)
	}

	var t = new(TaskReq)
	if err = json.Unmarshal(bodyBytes, t); err != nil {
		return nil, err
	}
	if err = t.Verification(); err != nil {
		return nil, err
	}

	var tw = t.To()
	if _, ok := c.store.Load(tw.ID); ok {
		return nil, server.StatusCode(http.StatusConflict)
	}

	c.w.ExecTask(tw)

	return tw.ID, server.StatusCode(http.StatusCreated)
}

// Delete, "/v1/task/{id}" отмена задания. {id}
func (c *Task) Delete(req *http.Request) (any, error) {
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
