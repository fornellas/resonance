package host

import (
	"context"
	"errors"
	"fmt"
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

func (s Ssh) Remove(ctx context.Context, name string) error {
	return errors.New("TODO Ssh.Remove")
}

func (s Ssh) Run(ctx context.Context, cmd Cmd) (WaitStatus, string, string, error) {
	return WaitStatus{}, "", "", errors.New("TODO Ssh.Run")
}

func (s Ssh) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	return fmt.Errorf("TODO Ssh.WriteFile")
}

func (s Ssh) String() string {
	return s.Hostname
}
