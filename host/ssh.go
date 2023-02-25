package host

import (
	"context"
	"errors"
	"os"
)

// Ssh interacts with a remote machine connecting to it via SSH protocol.
type Ssh struct {
	Hostname string
}

func (s Ssh) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	return nil, errors.New("TODO Ssh.Lstat")
}

func (s Ssh) ReadFile(ctx context.Context, name string) ([]byte, error) {
	return nil, errors.New("TODO Ssh.ReadFile")
}

func (s Ssh) Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {
	return WaitStatus{}, errors.New("TODO Ssh.Run")
}

func (s Ssh) String() string {
	return s.Hostname
}
