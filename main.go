//go:build !windows
// +build !windows

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"mediamagi.ru/win-file-agent/agent"
	"mediamagi.ru/win-file-agent/config"
	log1 "mediamagi.ru/win-file-agent/log"
	"mediamagi.ru/win-file-agent/script"
)

var (
	tPr      = flag.Bool("t", false, "Включить тест")
	urlPr    = flag.String("url", "http://91.220.62.199:8080/v1", "Ссылка на сервис")
	paramsPr = flag.String("params", `{"in_dir":"C:\\Users\\Administrator\\Downloads\\InDir","out_dir":"C:\\Users\\Administrator\\Downloads\\OutDir1","urls":[],"cmd":"ffmpeg.exe","args":["-i","{input}","-c:v","libx264","-b:v","500k","-c:a","copy","{output}"],"out_ext":"mp4"}`, "Параметры для вызова сервиса")
	csvPr    = flag.String("csv", "CS100files.txt", "Список urls видео для передачи")
	tcPr     = flag.Int("tc", 2, "Кол-во задач будет выполнять в параллель")
	fcPr     = flag.Int("fc", 3, "Кол-во файлов на скачивание")
)

func main() {
	flag.Parse()

	var ctx, cf = context.WithCancel(context.Background())
	defer cf()
	if *tPr {
		log1.Init(ctx, "test.log")

		script.TestRun(ctx, &script.Params{
			Url:       *urlPr,
			Req:       *paramsPr,
			Csv:       *csvPr,
			TaskCount: *tcPr,
			FileCount: *fcPr,
		})
		return
	}

	log1.Init(ctx, "")
	config.Init()

	var ag = agent.New(ctx)
	if err := ag.Start(ctx); err != nil {
		panic(err)
	}

	// блокируем до Ctrl+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	ag.OnStop()
}
