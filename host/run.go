package host

import (
	"context"
)

// Run implements Host interface by having all methods rely on an underlying Host.Run.
type Run struct {
	baseRun
	Host Host
}

func (r Run) Run(ctx context.Context, cmd Cmd) (WaitStatus, string, string, error) {
	return r.Host.Run(ctx, cmd)
}

func NewRun(host Host) Run {
	run := Run{
		Host: host,
	}
	run.baseRun.Host = host
	return run
}
