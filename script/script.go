package script

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"

	"mediamagi.ru/win-file-agent/server/controllers"
	"mediamagi.ru/win-file-agent/worker"
)

func TestRun(ctx context.Context, params *Params) {
	var wg sync.WaitGroup
	// кол-во задач в параллель + гоурутина с файлами
	wg.Add(params.TaskCount + 1)
	var fileCh = make(chan string, params.FileCount)

	for range params.TaskCount {
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				var reqData = new(controllers.TaskReq)
				if err := json.Unmarshal([]byte(params.Req), reqData); err != nil {
					panic(err)
				}

				for range params.FileCount {
					select {
					case <-ctx.Done():
						return
					case f := <-fileCh:
						reqData.Urls = append(reqData.Urls, f)
					}
				}

				taskID, err := postTask(params, *reqData)
				if err != nil {
					log.Printf("postTask err: %+v\n", err)
					continue
				}

				var taskState worker.StateCode
				for {
					select {
					case <-ctx.Done():
						return
					default:
					}

					task, err := getTask(params, taskID)
					if err != nil {
						log.Printf("getTask err: %+v\n", err)
						break
					}

					if taskState != task.State {
						// log
						taskState = task.State
						if taskState == worker.ERROR || taskState == worker.FINISH {
							log.Printf("stop ping Task: %+v\n", *task)
							break
						}

						log.Printf("NEW STATE Task: %+v\n", *task)
					}
				}
			}
		}()
	}

	go func(filePath string) {
		defer wg.Done()

		var lines = readFile(filePath)
		for {
			for _, file := range lines {
				select {
				case <-ctx.Done():
					return
				default:
				}
				fileCh <- file
			}
		}

	}(params.Csv)

	wg.Wait()

	log.Print("Test FINISH")
}

func readFile(filePath string) []string {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return lines
}

func postTask(params *Params, reqData controllers.TaskReq) (string, error) {
	buffer, err := json.Marshal(reqData)
	if err != nil {
		return "", err
	}

	urlL, err := url.Parse(params.Url)
	if err != nil {
		return "", err
	}
	urlL = urlL.JoinPath("task")

	req, err := http.NewRequest(http.MethodPost, urlL.String(), bytes.NewReader(buffer))
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		return "", err
	}

	buffer, err = io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var taskID string
	if err := json.Unmarshal(buffer, &taskID); err != nil {
		log.Printf("url: %s, taskID %s, taskJson: %s\n", urlL, taskID, string(buffer))
		return "", err
	}

	return taskID, nil
}

func getTask(params *Params, taskID string) (*worker.Task, error) {
	urlL, err := url.Parse(params.Url)
	if err != nil {
		return nil, err
	}
	urlL = urlL.JoinPath("task", taskID)

	req, err := http.NewRequest(http.MethodGet, urlL.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("url: %s, taskID %s\n", urlL, taskID)
		return nil, errors.New("not OK")
	}
	buffer, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var task = new(worker.Task)
	if err := json.Unmarshal(buffer, task); err != nil {
		log.Printf("url: %s, taskID %s, taskJson: %s\n", urlL, taskID, string(buffer))
		return nil, err
	}

	return task, nil
}
