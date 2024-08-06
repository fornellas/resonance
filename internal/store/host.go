package store

import (
	"context"

	"github.com/fornellas/resonance/host"
	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
	"github.com/fornellas/resonance/log"
)

// HostStorePath is the path where HostStore persists state.
var HostStorePath = "/var/lib/resonance"

// Implementation of Store that persists Blueprints at a Host at HostStorePath.
type HostStore struct {
	Host host.Host
}

// NewHostStore creates a new HostStore for given Host.
func NewHostStore(hst host.Host) *HostStore {
	return &HostStore{
		Host: hst,
	}
}

func (s *HostStore) GetLastBlueprint(ctx context.Context) (blueprintPkg.Blueprint, error) {
	logger := log.MustLogger(ctx).WithGroup("host_store")
	// ctx = log.WithLogger(ctx, logger)
	logger.Info("Computing blueprint")

	panic("TODO")
}

func (s *HostStore) Save(ctx context.Context, blueprint blueprintPkg.Blueprint) error {
	panic("TODO")
}
