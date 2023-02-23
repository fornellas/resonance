package state

import (
	"context"
	"errors"
)

type Local struct {
	PersistantState
	Path string
}

func (l Local) Load(ctx context.Context) (StateData, error) {
	return StateData{}, errors.New("TODO Local.Load")
}
func (l Local) Save(ctx context.Context, stateData StateData) error {
	return errors.New("TODO Local.Save")
}
