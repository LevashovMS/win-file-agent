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
	Files  []string
	Cmd    string
	Args   []string
	State  StateCode

	cmd *exec.Cmd
}
