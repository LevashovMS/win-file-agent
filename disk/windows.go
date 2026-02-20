//go:build windows
// +build windows

package disk

import "golang.org/x/sys/windows"

func GetFreeSpace(path string) (uint64, error) {
	var freeBytesAvailable uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64

	err := windows.GetDiskFreeSpaceEx(windows.StringToUTF16Ptr(path),
		&freeBytesAvailable, &totalNumberOfBytes, &totalNumberOfFreeBytes)
	println(freeBytesAvailable)

	return freeBytesAvailable, err
}
