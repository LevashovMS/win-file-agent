package log

import (
	"testing"

	"mediamagi.ru/win-file-agent/errors"
)

func TestWriter(t *testing.T) {
	Debug("Debug %d", 1)
	Info("Info %d", 2)
	Error("Error %+v", errors.Wrapf(testErr(), "scanln: %s", "male"))
}

func testErr() error {
	return errors.Errorf("test %d", 45)
}
