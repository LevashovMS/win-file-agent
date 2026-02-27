package worker

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"mediamagi.ru/win-file-agent/store"
)

func TestWorker(t *testing.T) {
	var ctx, cf = context.WithCancel(context.TODO())
	defer cf()

	var w = New(store.NewRam[string, *Task](ctx))
	if err := w.Run(ctx); err != nil {
		fmt.Printf("%+v\n", err)
		return
	}

	var task = defaultTask()
	w.ExecTask(task)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			time.Sleep(1 * time.Second)
			if t, ok := w.store.Load(task.ID); ok {
				//if t.State == PROCESS {
				//	if _, err := w.StopProc(task.ID); err != nil {
				//		fmt.Printf("%+v\n", err)
				//	}
				//	return
				//}
				if t.State == FINISH || t.State == ERROR {
					return
				}

				continue
			}
			return
		}

	}()

	wg.Wait()
	time.Sleep(1 * time.Second)
	fmt.Printf("task: %+v\n", task)
}

func defaultTask() *Task {
	return &Task{
		ID:     "111222",
		InDir:  "/home/max/Загрузки/tmp",
		OutDir: "/home/max/Загрузки/tmp_out",
		Urls: []string{
			"http://localhost:8088/test.mp4",
		},
		Cmd: "ffmpeg",
		Args: []string{
			"-i",
			"{input}",
			"-c:v",
			"libx264",
			"-b:v",
			"500k",
			"-c:a",
			"copy",
			"{output}",
		},
		//Files:  []string{"111_0"},
		OutExt: ".mp4",
	}
}
