package worker

import "os/exec"

const (
	INPUT  = "{input}"
	OUTPUT = "{output}"
)

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

func (c StateCode) String() string {
	switch c {
	case CREATE:
		return "CREATE"
	case DOWNLOAD:
		return "DOWNLOAD"
	case PROCESS:
		return "PROCESS"
	case SAVING:
		return "SAVING"
	case CANCEL:
		return "CANCEL"
	case FINISH:
		return "FINISH"
	case ERROR:
		return "ERROR"
	default:
	}
	return ""
}

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
