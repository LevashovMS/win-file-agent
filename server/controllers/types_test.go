package controllers

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestTypes(t *testing.T) {
	typeToJson[TaskReq]()

	var data = &TaskReq{
		InDir:  "in",
		OutDir: "out",
		Urls:   []string{""},
		Cmd:    "cmd",
		Args: []string{
			"{input}",
			"{output}",
		},
		OutExt: "mp4",
	}
	dataToJson(data)
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
