package store

import (
	"context"

	blueprintPkg "github.com/fornellas/resonance/blueprint"
	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

// Wraps a Store and log function calls.
type LoggingWrapper struct {
	store Store
}

func NewLoggingWrapper(store Store) *LoggingWrapper {
	return &LoggingWrapper{
		store: store,
	}
}

func (s *LoggingWrapper) SaveOriginalResource(ctx context.Context, resource resourcesPkg.Resource) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("SaveOriginalResource", "resource", resource)
	return s.store.SaveOriginalResource(ctx, resource)
}

func (s *LoggingWrapper) HasOriginalResource(ctx context.Context, resource resourcesPkg.Resource) (bool, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("HasOriginalResource", "resource", resource)
	return s.store.HasOriginalResource(ctx, resource)
}

func (s *LoggingWrapper) LoadOriginalResource(ctx context.Context, resource resourcesPkg.Resource) (resourcesPkg.Resource, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("LoadOriginalResource", "resource", resource)
	return s.store.LoadOriginalResource(ctx, resource)
}

func (s *LoggingWrapper) DeleteOriginalResource(ctx context.Context, resource resourcesPkg.Resource) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("DeleteOriginalResource", "resource", resource)
	return s.store.DeleteOriginalResource(ctx, resource)
}

func (s *LoggingWrapper) SaveLastBlueprint(ctx context.Context, blueprint *blueprintPkg.Blueprint) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("SaveLastBlueprint", "blueprint", blueprint)
	return s.store.SaveLastBlueprint(ctx, blueprint)
}

func (s *LoggingWrapper) LoadLastBlueprint(ctx context.Context) (*blueprintPkg.Blueprint, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("LoadLastBlueprint")
	return s.store.LoadLastBlueprint(ctx)
}

func (s *LoggingWrapper) SaveTargetBlueprint(ctx context.Context, blueprint *blueprintPkg.Blueprint) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("SaveTargetBlueprint", "blueprint", blueprint)
	return s.store.SaveTargetBlueprint(ctx, blueprint)
}

func (s *LoggingWrapper) HasTargetBlueprint(ctx context.Context) (bool, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("HasTargetBlueprint")
	return s.store.HasTargetBlueprint(ctx)
}

func (s *LoggingWrapper) LoadTargetBlueprint(ctx context.Context) (*blueprintPkg.Blueprint, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("LoadTargetBlueprint")
	return s.store.LoadTargetBlueprint(ctx)
}

func (s *LoggingWrapper) DeleteTargetBlueprint(ctx context.Context) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🗄️ Store")
	logger.Debug("DeleteTargetBlueprint")
	return s.store.DeleteTargetBlueprint(ctx)
}
