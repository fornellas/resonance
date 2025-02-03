package store

import (
	"context"

	resourcesPkg "github.com/fornellas/resonance/resources"
)

// Store defines an interface for storage of Blueprint.
type Store interface {
	// SaveOriginalResource saves given resource as its original state.
	SaveOriginalResource(context.Context, resourcesPkg.State) error

	// HasOriginalResource returns whether resource with given Id had a state previously with
	// SaveOriginalResource.
	HasOriginalResource(context.Context, resourcesPkg.State) (bool, error)

	// LoadOriginalResource returns the original state persisted with SaveOriginalResource for the
	// resource withgiven Id.
	LoadOriginalResource(context.Context, resourcesPkg.State) (resourcesPkg.State, error)

	// DeleteOriginalResource deletes a resource previously saved with SaveOriginalResource
	DeleteOriginalResource(context.Context, resourcesPkg.State) error

	// SaveLastBlueprint saves given Blueprint as the last.
	SaveLastBlueprint(context.Context, []resourcesPkg.State) error

	// LoadLastBlueprint returns the last Blueprint saved with SaveLastBlueprint.
	LoadLastBlueprint(context.Context) ([]resourcesPkg.State, error)

	// SaveTargetBlueprint saves given Blueprint as the target.
	SaveTargetBlueprint(context.Context, []resourcesPkg.State) error

	// HasTargetBlueprint returns whether a Blueprint was previously saved with SaveTargetBlueprint.
	HasTargetBlueprint(context.Context) (bool, error)

	// LoadTargetBlueprint returns the Blueprint saved with SaveTargetBlueprint.
	LoadTargetBlueprint(context.Context) ([]resourcesPkg.State, error)

	// DeleteTargetBlueprint deletes a target Blueprint previously saved with SaveTargetBlueprint
	DeleteTargetBlueprint(context.Context) error
}
