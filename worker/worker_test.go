package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mediamagi.ru/win-file-agent/config"
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

func TestExec(t *testing.T) {
	var ctx, cf = context.WithCancel(context.TODO())
	defer cf()
	config.InitWithPath("/config.json")
	var store = store.NewRam[string, *Task](ctx)
	var w = New(store)

	//ffmpeg -i /home/max/Загрузки/tmp/big-buck-bunny-1080p-30sec.mp4 -c:v libx264 -b:v 500k -c:a copy /home/max/Загрузки/tmp_out/output.mp4
	var task = &Task{
		ID:     "111",
		InDir:  "/home/max/Загрузки/tmp",
		OutDir: "/home/max/Загрузки/tmp_out",
		Cmd:    "ffmpeg",
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
		Files:  []string{"111_0"},
		OutExt: ".mp4",
	}
	go func() {
		var err = w.executeTask(ctx, task)
		fmt.Printf("err: %+v\n", err)
	}()

	time.Sleep(2 * time.Second)
	cf()
	time.Sleep(time.Second)
	fmt.Printf("task: %+v\n", task)
}

func TestFtp(t *testing.T) {
	var ctx, cf = context.WithCancel(context.TODO())
	defer cf()
	config.InitWithPath("/config.json")
	var store = store.NewRam[string, *Task](ctx)
	var w = New(store)

	//ffmpeg -i /home/max/Загрузки/tmp/big-buck-bunny-1080p-30sec.mp4 -c:v libx264 -b:v 500k -c:a copy /home/max/Загрузки/tmp_out/output.mp4
	var task = &Task{
		ID:    "111",
		InDir: "/home/max/Загрузки/tmp",
		//OutDir: "/home/max/Загрузки/tmp_out",
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
		Files:  []string{"CS100files.txt"},
		OutExt: ".mp4",
		ftp: &Ftp{
			Addr:  "",
			Login: "",
			Pass:  "",
		},
		saveToFtp: true,
	}
	go func() {
		var err = w.ftpStore(ctx, task)
		fmt.Printf("err: %+v\n", err)
	}()

	time.Sleep(2 * time.Second)
	cf()
	time.Sleep(time.Second)
	fmt.Printf("task: %+v\n", task)
}

func TestCmdStop(t *testing.T) {
	var ctx, cf = context.WithCancel(context.TODO())
	defer cf()
	config.InitWithPath("config/config.json")
	var store = store.NewRam[string, *Task](ctx)
	var w = New(store)

	var task = &Task{
		ID:     "111",
		InDir:  "/home/max/Загрузки/tmp",
		OutDir: "/home/max/Загрузки/tmp_out",
		Cmd:    "ffmpeg",
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
		Files:  []string{"111_0"},
		OutExt: ".mp4",
	}
	go func() {
		var err = w.executeTask(ctx, task)
		fmt.Printf("err: %+v\n", err)
	}()

	time.Sleep(1 * time.Second)
	w.StopProc("111")
	cf()
	time.Sleep(time.Second)
	fmt.Printf("task: %+v\n", task)
}
