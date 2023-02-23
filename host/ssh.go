package host

import (
	"context"
	"errors"
)

// Ssh interacts with a remote machine connecting to it via SSH protocol.
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
