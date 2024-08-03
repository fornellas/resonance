package store

import (
	"context"

	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
)

// Store defines an interface for storage of Blueprint.
type Store interface {
	// GetLastBlueprint returns the most recently saved Blueprint, or nil if unavailable.
	GetLastBlueprint(ctx context.Context) (blueprintPkg.Blueprint, error)
	// Save given Blueprint.
	Save(ctx context.Context, blueprint blueprintPkg.Blueprint) error
}
