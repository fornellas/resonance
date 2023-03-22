package host

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
)

// Ssh interacts with a remote machine connecting to it via SSH protocol.
type Ssh struct {
	Hostname string
}

func (s Ssh) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	return errors.New("TODO Ssh.Chmod")
}

func (s Ssh) Chown(ctx context.Context, name string, uid, gid int) error {
	return errors.New("TODO Ssh.Chown")
}

func (s Ssh) Lookup(ctx context.Context, username string) (*user.User, error) {
	return nil, errors.New("TODO Ssh.LookupId")
}

func (s Ssh) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	return nil, errors.New("TODO Ssh.LookupGroup")
}

func (s Ssh) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	return nil, errors.New("TODO Ssh.Lstat")
}

func (s Ssh) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return errors.New("TODO Ssh.Mkdir")
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
