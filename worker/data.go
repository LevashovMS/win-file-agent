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
	ID string
	// request
	InDir  string
	OutDir string
	Urls   []string
	Cmd    string
	Args   []string
	OutExt string
	// processing
	Files []string
	State StateCode
	Msg   string

	cmd *exec.Cmd
}
