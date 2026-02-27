package worker

import (
	"os/exec"
	"path/filepath"

	"mediamagi.ru/win-file-agent/config"
)

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

	cmd       *exec.Cmd `json:"-"`
	ftp       *Ftp      `json:"-"`
	saveToFtp bool      `json:"-"`
}

func (c *Task) SaveToFtp(ftp *Ftp) {
	c.saveToFtp = true
	c.ftp = ftp
}

func (c *Task) GetOutDir() string {
	if len(c.OutDir) != 0 {
		return c.OutDir
	}
	return config.Load().TmpDir
}

func (c *Task) GetOutPath(fileName string) string {
	var filePath = filepath.Join(c.GetOutDir(), fileName)
	return filePath + c.OutExt
}

type Ftp struct {
	Addr  string `json:"addr"`
	Login string `json:"login"`
	Pass  string `json:"pass"`
}
