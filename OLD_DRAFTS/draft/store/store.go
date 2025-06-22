package store

import (
	"context"

	resourcePkg "github.com/fornellas/resonance/draft/resource"
)

type Store interface {
	SaveOriginal(context.Context, resourcePkg.State) error
	HasOriginalId(context.Context, *resourcePkg.Id) (bool, error)
	GetOriginal(context.Context, *resourcePkg.Id) (resourcePkg.State, error)
	ListOriginalIds(context.Context) ([]*resourcePkg.Id, error)
	DeleteOriginal(context.Context, *resourcePkg.Id) error

	Stage(context.Context, []resourcePkg.State) error
	HasStaged(context.Context) (bool, error)

	Commit(context.Context) error
	GetCommitted(context.Context, *resourcePkg.Id)
	ListCommittedIds(context.Context) ([]*resourcePkg.Id, error)

	// Close Store and release exclusive lock
	Close(context.Context)
}
