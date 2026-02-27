//go:build !windows
// +build !windows

package disk

import (
	"golang.org/x/sys/unix"
	"mediamagi.ru/win-file-agent/errors"
)

func GetFreeSpace(path string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, errors.WithStack(err)
	}

	// Available blocks * size per block = available space in bytes
	println(stat.Bavail * uint64(stat.Bsize))
	return stat.Bavail * uint64(stat.Bsize), nil
}
