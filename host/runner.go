package host

import (
	"context"
)

// FIXME Runner is not needed, only useful to test baseRun

// Runner implements Host interface by having all methods rely on an underlying Host.Run.
type Runner struct {
	baseRun
	Host Host
}

func (r Runner) Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {
	return r.Host.Run(ctx, cmd)
}

func (r Runner) String() string {
	return r.Host.String()
}

func (r Runner) Close() error {
	return r.Host.Close()
}

func NewRunner(host Host) Runner {
	run := Runner{
		Host: host,
	}
	run.baseRun.Host = host
	return run
}
