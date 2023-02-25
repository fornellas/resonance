package host

import (
	"context"
	"errors"
	"os"
)

// Local interacts with the local machine running the code.
type Local struct{}

func (l Local) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	return os.Lstat(name)
}

func (l Local) ReadFile(ctx context.Context, name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (l Local) Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {
	return WaitStatus{}, errors.New("TODO Local.Run")
}

func (l Local) String() string {
	return "localhost"
}
