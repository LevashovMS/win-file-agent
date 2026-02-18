package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mediamagi.ru/win-file-agent/store"
)

func TestUrls(t *testing.T) {
	var ctx, cf = context.WithCancel(context.TODO())
	defer cf()
	var store = store.NewRam[string, *Task](ctx)
	var w = New(store)

	var task = &Task{
		ID:    "111",
		InDir: "/home/max/Загрузки/tmp",
		Urls: []string{
			"https://github.com/chthomos/video-media-samples/blob/997cb58f16bc3433652506910734be75bc64d768/big-buck-bunny-1080p-30sec.mp4",
		},
	}
	go func() {
		var err = w.downloadFiles(ctx, task)
		fmt.Printf("err: %+v\n", err)
	}()

	time.Sleep(2 * time.Second)
	cf()
	time.Sleep(time.Second)
	fmt.Printf("task: %+v\n", task)
}
