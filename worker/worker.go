package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mediamagi.ru/win-file-agent/config"
	"mediamagi.ru/win-file-agent/log"
	"mediamagi.ru/win-file-agent/store"
)

type Worker struct {
	cancel       context.CancelFunc
	count        int
	queue        int
	wg           sync.WaitGroup // воркер‑пул
	taskQueue    chan *Task
	store        store.Store[string, *Task]
	storeProc    store.Store[string, context.CancelFunc]
	shutdownOnce sync.Once
}

func New(storeT store.Store[string, *Task]) *Worker {
	var cfg = config.Load()
	var workerCount = cfg.WorkerCount
	if workerCount < 1 {
		workerCount = 4
	}
	if workerCount > 50 {
		workerCount = 16
	}
	var workerQueue = cfg.WorkerQueue
	if workerQueue < 1 {
		workerQueue = 10
	}

	return &Worker{
		count:     workerCount,
		taskQueue: make(chan *Task, workerQueue),
		store:     storeT,
		storeProc: store.NewRam[string, context.CancelFunc](context.TODO()),
	}
}

// Run запускает все компоненты
func (c *Worker) Run(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)
	// Запускаем воркер‑пул
	for range c.count {
		c.wg.Add(1)
		go c.workerLoop(ctx)
	}

	return nil
}

func (c *Worker) Stop() {
	c.shutdownOnce.Do(func() {
		log.Info("Shutdown requested")

		// 1) Сигналируем всему: отменяем контекст
		c.cancel()

		// 3) Ожидаем завершения воркеров
		done := make(chan struct{})
		go func() { c.wg.Wait(); close(done) }()

		select {
		case <-done:
			// всё ок
		case <-time.After(10 * time.Second):
			log.Info("Workers didn’t finish in time – force kill")
		}

		// 4) Принудительно завершаем «живающие» внешние процессы
		c.stopAllChildProcesses()
	})
}

func (c *Worker) ExecTask(t *Task) {
	c.store.Store(t.ID, t)
	c.taskQueue <- t
}

func (c *Worker) StopProc(key string) (bool, error) {
	var v, ok = c.store.Load(key)
	if !ok {
		return ok, nil
	}

	c.stopProc(key, v)
	return false, nil
}

func (c *Worker) workerLoop(ctx context.Context) {
	defer c.wg.Done()
	for {
		select {
		case <-ctx.Done():
			// Ожидаем завершения текущей задачи (если нужно)
			return
		case task := <-c.taskQueue:
			func() {
				var ctxPrc, cf = context.WithCancel(ctx)
				c.storeProc.Store(task.ID, cf)
				defer func() {
					c.storeProc.Delete(task.ID)
					clearFolders(task)
				}()

				var handlers = map[StateCode]workerHandler{
					DOWNLOAD: downloadFiles,
					PROCESS:  executeTask,
				}
				if task.saveToFtp {
					handlers[SAVING] = ftpStore
				}
				for state, handler := range handlers {
					c.setState(task.ID, state)
					if err := handler(ctxPrc, task); err != nil {
						log.Error("Task %s %s error: %+v", task.ID, state, err)
						c.setState(task.ID, ERROR, err)
						return
					}
				}

				log.Info("Task %s finished successfully", task.ID)
				c.setState(task.ID, FINISH)
			}()
		}
	}
}

func (c *Worker) stopAllChildProcesses() {
	c.store.Range(c.stopProc)
}

func (c *Worker) stopProc(key string, task *Task) bool {
	if cf, ok := c.storeProc.Load(key); ok {
		cf()
	}

	if task.State != PROCESS {
		return true
	}

	var cmd = task.cmd
	if cmd == nil {
		log.Error("Task %s cmd == nil", key)
		return true
	}
	// 1) Если процесс уже завершён – skip
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return true
	}

	// 2) Отправляем graceful‑kill (Ctrl+C) – но для cmd.exe/PowerShell это не всегда работает.
	// Лучше сразу kill
	if err := cmd.Process.Kill(); err != nil {
		log.Error("Failed to kill child %s: %+v", key, err)
	} else {
		log.Info("Killed child %s", key)
	}

	return true
}

func (c *Worker) setState(id string, state StateCode, errs ...error) {
	if task, ok := c.store.Load(id); ok {
		task.State = state
		if state == ERROR {
			task.Msg = fmt.Sprintf("%s", errs)
			c.store.SetTimeout(id, time.Now().Add(time.Minute))
		}
		if state == FINISH {
			c.store.SetTimeout(id, time.Now().Add(time.Minute))
		}
	}
}

func clearFolders(task *Task) {
	// удаляем файлы
	for _, fileName := range task.Files {
		var filePath = filepath.Join(task.InDir, fileName)
		if err := os.Remove(filePath); err != nil {
			log.Error("Task %s os.Remove error, filePath %s, err %+v\n", task.ID, filePath, err)
		}
		log.Debug("Task %s os.Remove successfully, filePath %s\n", task.ID, filePath)
		if task.saveToFtp {
			filePath = task.GetOutPath(fileName)
			if err := os.Remove(filePath); err != nil {
				log.Error("Task %s os.Remove error, filePath %s, err %+v\n", task.ID, filePath, err)
			} else {
				log.Debug("Task %s os.Remove successfully, filePath %s\n", task.ID, filePath)
			}
		}
	}
}
