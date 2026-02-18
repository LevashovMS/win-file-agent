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
	ID     string
	InDir  string
	OutDir string
	Urls   []string
	Files  []string
	Cmd    string
	Args   []string
	State  StateCode
	Msg    string

	cmd *exec.Cmd
}
