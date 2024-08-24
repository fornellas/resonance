package store

import (
	"context"

	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

// Store defines an interface for storage of Blueprint.
type Store interface {
	// SaveOriginalResource saves given resource as its original state.
	SaveOriginalResource(ctx context.Context, resource resourcesPkg.Resource) error

	// HasOriginalResource returns whether resource with given Id had a state previously with
	// SaveOriginalResource.
	HasOriginalResource(ctx context.Context, resource resourcesPkg.Resource) (bool, error)

	// LoadOriginalResource returns the original state persisted with SaveOriginalResource for the
	// resource withgiven Id.
	LoadOriginalResource(ctx context.Context, resource resourcesPkg.Resource) (resourcesPkg.Resource, error)

	// SaveLastBlueprint saves given Blueprint as the last.
	SaveLastBlueprint(ctx context.Context, blueprint *blueprintPkg.Blueprint) error

	// LoadLastBlueprint returns the last Blueprint saved with SaveLastBlueprint.
	LoadLastBlueprint(ctx context.Context) (*blueprintPkg.Blueprint, error)

	// SaveTargetBlueprint saves given Blueprint as the target.
	SaveTargetBlueprint(ctx context.Context, blueprint *blueprintPkg.Blueprint) error

	// HasTargetBlueprint returns whether a Blueprint was previously saved with SaveTargetBlueprint.
	HasTargetBlueprint(ctx context.Context) (bool, error)

	// LoadTargetBlueprint returns the Blueprint saved with SaveTargetBlueprint.
	LoadTargetBlueprint(ctx context.Context) (*blueprintPkg.Blueprint, error)

	// DeleteTargetBlueprint deletes a target Blueprint previously saved with SaveTargetBlueprint
	DeleteTargetBlueprint(ctx context.Context) error

	// TODO add locking interface
	// each client can generate its own UUID: if locked, and 100% sure there's no other active client for the same UUID (eg; no other process on the same machine), then safely remove lock
}
