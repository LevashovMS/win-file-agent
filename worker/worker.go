package worker

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"mediamagi.ru/win-file-agent/config"
	"mediamagi.ru/win-file-agent/ftp"
	"mediamagi.ru/win-file-agent/store"
)

type Worker struct {
	cancel       context.CancelFunc
	count        int
	queue        int
	wg           sync.WaitGroup // воркер‑пул
	taskQueue    chan *Task
	store        store.Store[string, *Task]
	shutdownOnce sync.Once
}

func New(store store.Store[string, *Task]) *Worker {
	var cfg = config.Cfg.Load()
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
		store:     store,
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
				defer func() {
					// удаляем файлы
					for _, fileName := range task.Files {
						var filePath = filepath.Join(task.InDir, fileName)
						os.Remove(filePath)
						if task.saveToFtp {
							filePath = filepath.Join(task.GetOutDir(), fileName)
							os.Remove(filePath)
						}
					}
				}()

				if err := c.downloadFiles(ctx, task); err != nil {
					log.Printf("Task %s downloadFiles error: %v", task.ID, err)
					c.setState(task.ID, ERROR, err)
					return
				}
				if err := c.executeTask(ctx, task); err != nil {
					log.Printf("Task %s executeTask error: %v", task.ID, err)
					c.setState(task.ID, ERROR, err)
					return
				}
				if err := c.ftpStore(ctx, task); err != nil {
					log.Printf("Task %s ftpStore error: %v", task.ID, err)
					c.setState(task.ID, ERROR, err)
					return
				}

				log.Printf("Task %s finished successfully", task.ID)
				c.setState(task.ID, FINISH)
			}()
		}
	}
}

func (c *Worker) downloadFiles(ctx context.Context, task *Task) error {
	c.setState(task.ID, DOWNLOAD)
	for idx, urlStr := range task.Urls {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var fileName = fmt.Sprintf("%s_%d", task.ID, idx)
		// 1. Get the data from the URL
		resp, err := http.Get(urlStr)
		if err != nil {
			return err
		}
		// Ensure the response body is closed after the function returns
		defer resp.Body.Close()

		var filepath = filepath.Join(task.InDir, fileName)
		// 2. Create the local file
		out, err := os.Create(filepath)
		if err != nil {
			return err
		}
		// Ensure the file is closed after the function returns
		defer out.Close()

		// 3. Stream the response body to the file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err
		}
		task.Files = append(task.Files, fileName)

		log.Printf("Task %s Downloaded file to %s\n", task.ID, filepath)
	}
	return nil
}

func (c *Worker) executeTask(ctx context.Context, task *Task) error {
	c.setState(task.ID, PROCESS)

	for _, fileName := range task.Files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var args = make([]string, len(task.Args))
		for idx, it := range task.Args {
			if it == INPUT {
				var filePath = filepath.Join(task.InDir, fileName)
				args[idx] = filePath
				continue
			}
			if it == OUTPUT {
				var filePath = filepath.Join(task.GetOutDir(), fileName)
				args[idx] = filePath + task.OutExt
				continue
			}

			args[idx] = task.Args[idx]
		}
		fmt.Printf("args: %v\n", args)

		// пример: ffmpeg -i input.mp4 -c:v libx264 -b:v 500k -c:a copy output.mp4
		cmd := exec.CommandContext(ctx, task.Cmd, args...)
		// настройка
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// запускаем
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("err %+v cmd %+v, args %+v", err, task.Cmd, args)
		}
		// ждём завершения
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("err %+v cmd %+v, args %+v", err, task.Cmd, args)
		}

		log.Printf("Task %s exec.Command successfully, cmd %+v, args %+v\n", task.ID, task.Cmd, args)
	}

	return nil
}

func (c *Worker) ftpStore(ctx context.Context, task *Task) error {
	c.setState(task.ID, SAVING)
	if !task.saveToFtp {
		return nil
	}

	ftpClient, err := ftp.Dial(task.Ftp.Addr, ftp.DialWithContext(ctx))
	if err != nil {
		return fmt.Errorf("ftp.Dial Task %s err %+v Addr %s", task.ID, err, task.Ftp.Addr)
	}
	defer func() {
		if err := ftpClient.Quit(); err != nil {
			log.Printf("ftpClient.Quit() err: %+v\n", err)
		}
	}()

	err = ftpClient.Login(task.Ftp.Login, task.Ftp.Pass)
	if err != nil {
		return fmt.Errorf("ftpClient.Login Task %s err %+v Login %s Pass %s", task.ID, err, task.Ftp.Login, task.Ftp.Pass)
	}

	for _, fileName := range task.Files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err = func() error {
			var filePath = filepath.Join(task.GetOutDir(), fileName)
			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("os.Open Task %s err %+v filePath %s", task.ID, err, filePath)
			}
			defer file.Close()

			err = ftpClient.Stor(fileName, file)
			if err != nil {
				return fmt.Errorf("ftpClient.Stor Task %s err %+v fileName %s filePath %s", task.ID, err, fileName, filePath)
			}

			return nil
		}()
	}
	return nil
}

func (c *Worker) stopAllChildProcesses() {
	c.store.Range(c.stopProc)
}

func (c *Worker) stopProc(key string, task *Task) bool {
	if task.State != PROCESS {
		return true
	}

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

func (c *Worker) setState(id string, state StateCode, errs ...error) {
	if task, ok := c.store.Load(id); ok {
		task.State = state
		if state == ERROR {
			task.Msg = fmt.Sprintf("%v", errs)
			c.store.SetTimeout(id, time.Now().Add(time.Minute))
		}
		if state == FINISH {
			c.store.SetTimeout(id, time.Now().Add(time.Minute))
		}
	}
}
