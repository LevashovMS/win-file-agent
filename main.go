//go:build !windows
// +build !windows

package main

import (
	"context"
	"os"
	"os/signal"

	"mediamagi.ru/win-file-agent/agent"
)

func main() {
	var ctx, cf = context.WithCancel(context.Background())
	defer cf()

	var ag = agent.New()
	if err := ag.Start(ctx); err != nil {
		panic(err)
	}

	// блокируем до Ctrl+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	ag.OnStop()
}
