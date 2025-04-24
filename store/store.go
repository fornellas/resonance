package store

import (
	"context"

	blueprintPkg "github.com/fornellas/resonance/blueprint"
	"github.com/fornellas/resonance/log"
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

	// DeleteOriginalResource deletes a resource previously saved with SaveOriginalResource
	DeleteOriginalResource(ctx context.Context, resource resourcesPkg.Resource) error

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
}

// Wraps a Store and log function calls.
type LoggingStore struct {
	store Store
}

func NewLoggingStore(store Store) Store {
	return &LoggingStore{
		store: store,
	}
}

func (s *LoggingStore) SaveOriginalResource(ctx context.Context, resource resourcesPkg.Resource) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("SaveOriginalResource", "resource", resource)
	return s.store.SaveOriginalResource(ctx, resource)
}

func (s *LoggingStore) HasOriginalResource(ctx context.Context, resource resourcesPkg.Resource) (bool, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("HasOriginalResource", "resource", resource)
	return s.store.HasOriginalResource(ctx, resource)
}

func (s *LoggingStore) LoadOriginalResource(ctx context.Context, resource resourcesPkg.Resource) (resourcesPkg.Resource, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("LoadOriginalResource", "resource", resource)
	return s.store.LoadOriginalResource(ctx, resource)
}

func (s *LoggingStore) DeleteOriginalResource(ctx context.Context, resource resourcesPkg.Resource) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("DeleteOriginalResource", "resource", resource)
	return s.store.DeleteOriginalResource(ctx, resource)
}

func (s *LoggingStore) SaveLastBlueprint(ctx context.Context, blueprint *blueprintPkg.Blueprint) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("SaveLastBlueprint", "blueprint", blueprint)
	return s.store.SaveLastBlueprint(ctx, blueprint)
}

func (s *LoggingStore) LoadLastBlueprint(ctx context.Context) (*blueprintPkg.Blueprint, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("LoadLastBlueprint")
	return s.store.LoadLastBlueprint(ctx)
}

func (s *LoggingStore) SaveTargetBlueprint(ctx context.Context, blueprint *blueprintPkg.Blueprint) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("SaveTargetBlueprint", "blueprint", blueprint)
	return s.store.SaveTargetBlueprint(ctx, blueprint)
}

func (s *LoggingStore) HasTargetBlueprint(ctx context.Context) (bool, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("HasTargetBlueprint")
	return s.store.HasTargetBlueprint(ctx)
}

func (s *LoggingStore) LoadTargetBlueprint(ctx context.Context) (*blueprintPkg.Blueprint, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("LoadTargetBlueprint")
	return s.store.LoadTargetBlueprint(ctx)
}

func (s *LoggingStore) DeleteTargetBlueprint(ctx context.Context) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("DeleteTargetBlueprint")
	return s.store.DeleteTargetBlueprint(ctx)
}
