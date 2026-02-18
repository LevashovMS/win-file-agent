//go:build windows
// +build windows

package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
)

type AgentWindows struct {
	ctx context.Context
	cf  context.CancelFunc
	a   *Agent
}

func NewAgentWindows() *AgentWindows {
	var ctx, cf = context.WithCancel(context.Background())
	return &AgentWindows{
		ctx: ctx,
		cf:  cf,
		a:   New(ctx),
	}
}

func (a *AgentWindows) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	fasttick := time.Tick(500 * time.Millisecond)
	slowtick := time.Tick(2 * time.Second)
	tick := fasttick
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case <-tick:
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				// golang.org/x/sys/windows/svc.TestExample is verifying this output.
				testOutput := strings.Join(args, "-")
				testOutput += fmt.Sprintf("-%d", c.Context)
				a.a.OnStop()
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				tick = slowtick
				a.a.OnStop()
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				tick = fasttick
				a.a.Start(a.ctx)
			default:
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}
