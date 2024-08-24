package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
	"github.com/fornellas/resonance/internal/diff"
	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

func getTestBlueprint(t *testing.T, ctx context.Context) *blueprintPkg.Blueprint {
	resources := resourcesPkg.Resources{
		&resourcesPkg.File{
			Path: "/tmp/foo",
			Uid:  123,
			Gid:  456,
		},
		&resourcesPkg.APTPackage{
			Package: "foo",
			Version: "1.2.3",
		},
	}
	blueprint, err := blueprintPkg.NewBlueprintFromResources(ctx, resources)
	require.NoError(t, err)
	return blueprint
}

func testStore(t *testing.T, store Store) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	t.Run("OriginalResource", func(t *testing.T) {
		resources := resourcesPkg.Resources{
			&resourcesPkg.File{
				Path: "/tmp/foo",
				Uid:  123,
				Gid:  456,
			},
			&resourcesPkg.APTPackage{
				Package: "foo",
				Version: "1.2.3",
			},
		}
		for _, resource := range resources {
			hasOriginalResource, err := store.HasOriginalResource(ctx, resource)
			require.NoError(t, err)
			require.False(t, hasOriginalResource)

			require.NoError(t, store.SaveOriginalResource(ctx, resource))

			hasOriginalResource, err = store.HasOriginalResource(ctx, resource)
			require.NoError(t, err)
			require.True(t, hasOriginalResource)

			loadedResource, err := store.LoadOriginalResource(ctx, resourcesPkg.NewResourceWithSameId(resource))
			require.NoError(t, err)
			require.Empty(t, diff.DiffAsYaml(resource, loadedResource))
		}

	})

	t.Run("LastBlueprint", func(t *testing.T) {
		nilBlueprint, err := store.LoadLastBlueprint(ctx)
		require.NoError(t, err)
		require.Nil(t, nilBlueprint)

		blueprint := getTestBlueprint(t, ctx)

		require.NoError(t, store.SaveLastBlueprint(ctx, blueprint))

		loadedBlueprint, err := store.LoadLastBlueprint(ctx)
		require.NoError(t, err)
		require.Empty(t, diff.DiffAsYaml(blueprint, loadedBlueprint))
	})

	t.Run("LastBlueprint", func(t *testing.T) {
		hasTargetBlueprint, err := store.HasTargetBlueprint(ctx)
		require.NoError(t, err)
		require.False(t, hasTargetBlueprint)

		blueprint := getTestBlueprint(t, ctx)

		require.NoError(t, store.SaveTargetBlueprint(ctx, blueprint))
		require.NoError(t, err)

		hasTargetBlueprint, err = store.HasTargetBlueprint(ctx)
		require.NoError(t, err)
		require.True(t, hasTargetBlueprint)

		loadedBlueprint, err := store.LoadTargetBlueprint(ctx)
		require.NoError(t, err)
		require.Empty(t, diff.DiffAsYaml(blueprint, loadedBlueprint))

		require.NoError(t, store.DeleteTargetBlueprint(ctx))

		hasTargetBlueprint, err = store.HasTargetBlueprint(ctx)
		require.NoError(t, err)
		require.False(t, hasTargetBlueprint)
	})
}
