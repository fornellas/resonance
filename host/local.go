package host

import (
	"context"
	"errors"
)

// Local interacts with the local machine running the code.
type Local struct {
	Host
}

func (l Local) Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {
	return WaitStatus{}, errors.New("TODO Local.Run")
}

func (l Local) String() string {
	return "localhost"
}
