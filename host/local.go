package host

import (
	"context"
	"errors"
)

type Local struct {
	Host
}

func (l Local) Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {
	return WaitStatus{}, errors.New("TODO Local.Run")
}

func (l Local) String() string {
	return "localhost"
}
