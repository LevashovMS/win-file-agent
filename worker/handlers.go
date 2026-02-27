package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"mediamagi.ru/win-file-agent/errors"
	"mediamagi.ru/win-file-agent/ftp"
	"mediamagi.ru/win-file-agent/log"
)

type workerHandler func(ctx context.Context, task *Task) error

func downloadFiles(ctx context.Context, task *Task) error {
	for idx, urlStr := range task.Urls {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var fileName = fmt.Sprintf("%s_%d", task.ID, idx)
		// 1. Get the data from the URL
		var req, err = http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
		if err != nil {
			return errors.Errorf("fileName %s, urlStr %s, err %+v", fileName, urlStr, err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.Errorf("fileName %s, urlStr %s, err %+v", fileName, urlStr, err)
		}
		// Ensure the response body is closed after the function returns
		defer resp.Body.Close()

		var filepath = filepath.Join(task.InDir, fileName)
		// 2. Create the local file
		out, err := os.Create(filepath)
		if err != nil {
			return errors.Errorf("fileName %s, urlStr %s, err %+v", fileName, urlStr, err)
		}
		// Ensure the file is closed after the function returns
		defer out.Close()

		// фиксируем имя файла для удаления до самого копирования.
		task.Files = append(task.Files, fileName)
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return errors.Errorf("fileName %s, urlStr %s, err %+v", fileName, urlStr, err)
		}

		log.Debug("Task %s url %s Downloaded file to %s\n", task.ID, urlStr, filepath)
	}
	return nil
}

func executeTask(ctx context.Context, task *Task) error {
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
				args[idx] = task.GetOutPath(fileName)
				continue
			}

			args[idx] = task.Args[idx]
		}
		fmt.Printf("args: %v\n", args)

		// пример: ffmpeg -i input.mp4 -c:v libx264 -b:v 500k -c:a copy output.mp4
		cmd := exec.CommandContext(ctx, task.Cmd, args...)
		// настройка
		var buffer = new(bytes.Buffer)
		cmd.Stdout = os.Stdout
		//cmd.Stderr = os.Stderr
		cmd.Stderr = buffer
		task.cmd = cmd

		// запускаем
		if err := cmd.Start(); err != nil {
			return errors.Errorf("err (%s), cmdErr %s, cmd %s, args %+v", err, buffer, task.Cmd, args)
		}
		// ждём завершения
		if err := cmd.Wait(); err != nil {
			return errors.Errorf("err (%s), cmdErr %s, cmd %s, args %+v", err, buffer, task.Cmd, args)
		}

		log.Debug("Task %s exec.Command successfully, cmd %+v, args %+v\n", task.ID, task.Cmd, args)
	}

	return nil
}

func ftpStore(ctx context.Context, task *Task) error {
	if !task.saveToFtp {
		return nil
	}

	ftpClient, err := ftp.Dial(task.ftp.Addr, ftp.DialWithContext(ctx))
	if err != nil {
		return errors.Errorf("ftp.Dial Task %s err %+v Addr %s", task.ID, err, task.ftp.Addr)
	}
	defer func() {
		if err := ftpClient.Quit(); err != nil {
			log.Error("ftpClient.Quit() err: %+v", err)
		}
	}()

	err = ftpClient.Login(task.ftp.Login, task.ftp.Pass)
	if err != nil {
		return errors.Errorf("ftpClient.Login Task %s err %+v Login %s Pass %s", task.ID, err, task.ftp.Login, task.ftp.Pass)
	}

	for _, fileName := range task.Files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err = func() error {
			var filePath = task.GetOutPath(fileName)
			file, err := os.Open(filePath)
			if err != nil {
				return errors.Errorf("os.Open Task %s err %+v filePath %s", task.ID, err, filePath)
			}
			defer file.Close()

			err = ftpClient.Stor(fileName, file)
			if err != nil {
				return errors.Errorf("ftpClient.Stor Task %s err %+v fileName %s filePath %s", task.ID, err, fileName, filePath)
			}

			log.Debug("Task %s ftpStore successfully, filePath %s\n", task.ID, filePath)
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
