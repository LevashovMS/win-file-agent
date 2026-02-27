package worker

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestUrls(t *testing.T) {
	var ctx, cf = context.WithCancel(context.TODO())
	defer cf()

	var wg sync.WaitGroup
	var task = defaultTask()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err = downloadFiles(ctx, task)
		fmt.Printf("err: %s\n", err)
	}()

	time.Sleep(1000 * time.Millisecond)
	cf()
	wg.Wait()
	fmt.Printf("task: %+v\n", task)
}

func TestExec(t *testing.T) {
	var ctx, cf = context.WithCancel(context.TODO())
	defer cf()

	//ffmpeg -i /home/max/Загрузки/tmp/big-buck-bunny-1080p-30sec.mp4 -c:v libx264 -b:v 500k -c:a copy /home/max/Загрузки/tmp_out/output.mp4
	var task = defaultTask()
	go func() {
		var err = executeTask(ctx, task)
		fmt.Printf("%s\n", err)
	}()

	//time.Sleep(2 * time.Second)
	//cf()
	time.Sleep(time.Second)
	fmt.Printf("task: %+v\n", task)
}

func TestFtp(t *testing.T) {
	var ctx, cf = context.WithCancel(context.TODO())
	defer cf()

	//ffmpeg -i /home/max/Загрузки/tmp/big-buck-bunny-1080p-30sec.mp4 -c:v libx264 -b:v 500k -c:a copy /home/max/Загрузки/tmp_out/output.mp4
	var task = defaultTask()
	go func() {
		var err = ftpStore(ctx, task)
		fmt.Printf("err: %+v\n", err)
	}()

	time.Sleep(2 * time.Second)
	cf()
	time.Sleep(time.Second)
	fmt.Printf("task: %+v\n", task)
}
