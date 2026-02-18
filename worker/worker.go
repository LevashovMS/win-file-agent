package worker

import (
	"context"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"mediamagi.ru/win-file-agent/store"
)

type Worker struct {
	cancel       context.CancelFunc
	wg           sync.WaitGroup // воркер‑пул
	taskQueue    chan *Task
	store        store.Store[string, *Task]
	shutdownOnce sync.Once
}

func New(store store.Store[string, *Task]) *Worker {
	return &Worker{
		taskQueue: make(chan *Task),
		store:     store,
	}
}

// Run запускает все компоненты
func (c *Worker) Run(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)
	// Запускаем воркер‑пул
	c.wg.Add(1)
	go c.workerLoop(ctx)

	return nil
}

func (c *Worker) Stop() {
	c.shutdownOnce.Do(func() {
		log.Println("Shutdown requested")

		// 1) Сигналируем всему: отменяем контекст
		c.cancel()

		// 3) Ожидаем завершения воркеров
		done := make(chan struct{})
		go func() { c.wg.Wait(); close(done) }()

		select {
		case <-done:
			// всё ок
		case <-time.After(10 * time.Second):
			log.Println("Workers didn’t finish in time – force kill")
		}

		// 4) Принудительно завершаем «живающие» внешние процессы
		c.stopAllChildProcesses()
	})
}

func (c *Worker) RunProc(t *Task) {
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
			c.executeTask(ctx, task)
		}
	}
}

func (c *Worker) executeTask(ctx context.Context, task *Task) {
	// создаём exec.Cmd
	// пример: ffmpeg -i input.mp4 -c:v libx264 -b:v 500k -c:a copy output.mp4
	cmd := exec.CommandContext(ctx, task.Cmd, task.Args...)
	// настройка
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// сохраняем в map для последующего kill
	c.store.Store(task.ID, task)
	c.setState(task.ID, PROCESS)

	// запускаем
	if err := cmd.Start(); err != nil {
		log.Printf("Task %s start error: %v", task.ID, err)
		c.setState(task.ID, ERROR)
		return
	}

	// ждём завершения в отдельной горутине
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("Task %s finished with error: %v", task.ID, err)
			c.setState(task.ID, ERROR)
		} else {
			log.Printf("Task %s finished successfully", task.ID)
			c.store.Delete(task.ID)
		}
	}()
}

func (c *Worker) stopAllChildProcesses() {
	c.store.Range(c.stopProc)
}

func (c *Worker) stopProc(key string, task *Task) bool {
	var cmd = task.cmd
	// 1) Если процесс уже завершён – skip
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return true
	}

	// 2) Отправляем graceful‑kill (Ctrl+C) – но для cmd.exe/PowerShell это не всегда работает.
	// Лучше сразу kill
	if err := cmd.Process.Kill(); err != nil {
		log.Printf("Failed to kill child %s: %v", key, err)
	} else {
		log.Printf("Killed child %s", key)
	}

	return true
}

func (c *Worker) setState(id string, state StateCode) {
	if task, ok := c.store.Load(id); ok {
		task.State = state
	}
}
