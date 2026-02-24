package controllers

import (
	"encoding/json"
	"fmt"
	"testing"

	"mediamagi.ru/win-file-agent/worker"
)

func TestTypes(t *testing.T) {
	typeToJson[TaskReq]()

	var data = &TaskReq{
		InDir: "in",
		//OutDir: "out",
		Urls: []string{""},
		Cmd:  "cmd",
		Args: []string{
			"{input}",
			"{output}",
		},
		OutExt: "mp4",
		Ftp: &worker.Ftp{
			Addr:  "addr",
			Login: "login",
			Pass:  "pass",
		},
	}
	dataToJson(data)

	var data2 = data.ToWTask()
	data2.Files = []string{""}
	data2.State = worker.CREATE
	data2.Msg = "msg"
	dataToJson(data2)
}

func typeToJson[T any]() {
	var t = new(T)
	var buffer, _ = json.Marshal(t)
	fmt.Printf("%v\n", string(buffer))
}

func dataToJson[T any](data T) {
	var buffer, _ = json.Marshal(data)
	fmt.Printf("%v\n", string(buffer))
}
