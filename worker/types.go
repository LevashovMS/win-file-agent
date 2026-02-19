package worker

import "os/exec"

type StateCode int8

const (
	CREATE StateCode = iota
	DOWNLOAD
	PROCESS
	SAVING
	CANCEL
	FINISH
	ERROR StateCode = 127
)

type Task struct {
	ID string `json:"id"`
	// request
	InDir  string   `json:"in_dir"`
	OutDir string   `json:"out_dir"`
	Urls   []string `json:"urls"`
	Cmd    string   `json:"cmd"`
	Args   []string `json:"args"`
	OutExt string   `json:"out_ext"`
	// processing
	Files []string  `json:"files"`
	State StateCode `json:"state"`
	Msg   string    `json:"msg"`

	cmd *exec.Cmd
}
