package state

import (
	"context"

	"github.com/fornellas/resonance/resource"
)

type PersistantState interface {
	Load(ctx context.Context) (resource.StateData, error)
	Save(ctx context.Context, stateData resource.StateData) error
}
