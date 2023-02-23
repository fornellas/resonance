package host

import (
	"context"
	"errors"
)

type Ssh struct {
	Host
	Hostname string
}

func (s Ssh) Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {
	return WaitStatus{}, errors.New("TODO Ssh.Run")
}

func (s Ssh) String() string {
	return s.Hostname
}
