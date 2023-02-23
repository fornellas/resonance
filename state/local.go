package state

import (
	"context"
	"errors"

	"github.com/fornellas/resonance/resource"
)

type Local struct {
	PersistantState
	Path string
}

func (l Local) Load(ctx context.Context) (resource.StateData, error) {
	return resource.StateData{}, errors.New("TODO Local.Load")
}
func (l Local) Save(ctx context.Context, stateData resource.StateData) error {
	return errors.New("TODO Local.Save")
}
