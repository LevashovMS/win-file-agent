//go:build windows
// +build windows

package main

import (
	"log"

	"golang.org/x/sys/windows/svc"

	log1 "mediamagi.ru/win-file-agent/1log"
	"mediamagi.ru/win-file-agent/agent"
)

const svcName = "FileAgent1"

func main() {
	defer log1.FileClose()

	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Failed to determine if we are running as a service: %v", err)
	}
	if isService {
		agent.RunService(svcName, false)
		return
	}

	agent.RunCommand(svcName)
}
