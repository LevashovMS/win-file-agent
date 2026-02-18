package agent

import (
	"context"
	"net/http"

	"mediamagi.ru/win-file-agent/server"
	"mediamagi.ru/win-file-agent/server/controllers"
	"mediamagi.ru/win-file-agent/store"
	"mediamagi.ru/win-file-agent/worker"
)

type Agent struct {
	s server.Server
	w *worker.Worker
}

func New() *Agent {
	var store = store.NewRam[string, *worker.Task]()
	var w = worker.New(store)
	var taskController = controllers.NewTask(store, w)
	// обычный запуск
	var s = server.New(
		server.Port(8080),
		server.Handler(http.MethodGet, "/v1/task/{id}", taskController.Get),
		server.Handler(http.MethodGet, "/v1/task", taskController.GetAll),
		server.Handler(http.MethodPost, "/v1/task", taskController.Create),
		server.Handler(http.MethodDelete, "/v1/task/{id}", taskController.Delete),
	)

	return &Agent{
		w: w,
		s: s,
	}
}

// Start запускает все компоненты
func (c *Agent) Start(ctx context.Context) error {
	if err := c.w.Run(ctx); err != nil {
		return err
	}

	if err := c.s.Run(ctx); err != nil {
		return err
	}
	return nil
}

// OnStop вызывается из Windows ServiceControlManager
func (c *Agent) OnStop() {
	c.w.Stop()
	c.s.Stop()
}
