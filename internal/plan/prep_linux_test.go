package plan

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
	iHostPkg "github.com/fornellas/resonance/internal/host"

	"github.com/fornellas/resonance/log"

	storePkg "github.com/fornellas/resonance/internal/store"
	resouresPkg "github.com/fornellas/resonance/resources"
)

func TestCreateAndStoreTargetBlueprint(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	host := iHostPkg.Local{}

	targetResources := resouresPkg.Resources{
		&resouresPkg.File{
			Path: "/bin",
			User: "root",
		},
		&resouresPkg.File{
			Path: "/lib",
			User: "root",
		},
	}

	targetBlueprint, err := CreateTargetBlueprint(
		ctx,
		host,
		targetResources,
	)
	require.NoError(t, err)

	require.Equal(t, resouresPkg.Resources{
		&resouresPkg.File{
			Path: "/bin",
			Uid:  0,
		},
		&resouresPkg.File{
			Path: "/lib",
			Uid:  0,
		},
	}, targetBlueprint.Resources())
}

func TestSaveOriginalResourcesState(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	host := iHostPkg.Local{}

	storePath := filepath.Join(t.TempDir(), "store")
	store := storePkg.NewHostStore(host, storePath)

	filePath := filepath.Join(t.TempDir(), "foo")
	fileContent := "foo"
	var fileMode uint32 = 0644
	err := host.WriteFile(ctx, filePath, []byte("foo"), fileMode)
	require.NoError(t, err)
	fileResource := &resouresPkg.File{
		Path:    filePath,
		Content: fileContent,
	}

	targetBlueprint, err := blueprintPkg.NewBlueprintFromResources(ctx, resouresPkg.Resources{
		fileResource,
	})
	require.NoError(t, err)

	err = SaveOriginalResourcesState(ctx, host, store, targetBlueprint)
	require.NoError(t, err)

	resource, err := store.LoadOriginalResource(ctx, &resouresPkg.File{
		Path: filePath,
	})
	require.NoError(t, err)
	require.Equal(t,
		&resouresPkg.File{
			Path:    filePath,
			Content: fileContent,
			Mode:    fileMode,
			Uid:     uint32(os.Getuid()),
			Gid:     uint32(os.Getgid()),
		},
		resource,
	)
}

func TestLoadOrCreateAndSaveLastBlueprintWithValidation(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	host := iHostPkg.Local{}

	storePath := filepath.Join(t.TempDir(), "store")
	store := storePkg.NewHostStore(host, storePath)

	filePath := filepath.Join(t.TempDir(), "foo")
	fileContent := "foo"
	var fileMode uint32 = 0644
	err := host.WriteFile(ctx, filePath, []byte("foo"), fileMode)
	require.NoError(t, err)
	fileResource := &resouresPkg.File{
		Path:    filePath,
		Content: fileContent,
	}

	targetBlueprint, err := blueprintPkg.NewBlueprintFromResources(ctx, resouresPkg.Resources{
		fileResource,
	})
	require.NoError(t, err)

	lastBlueprint, err := LoadOrCreateAndSaveLastBlueprintWithValidation(
		ctx,
		host,
		store,
		targetBlueprint,
	)
	require.NoError(t, err)
	require.Equal(t, len(targetBlueprint.Steps), len(lastBlueprint.Steps))
	for i, lastStep := range lastBlueprint.Steps {
		require.Equal(t, lastStep.DetailedString(), lastBlueprint.Steps[i].DetailedString())
	}

	loadedLastBlueprint, err := store.LoadLastBlueprint(ctx)
	require.NoError(t, err)
	require.Equal(t, len(lastBlueprint.Steps), len(loadedLastBlueprint.Steps))
	for i, lastStep := range lastBlueprint.Steps {
		require.Equal(t, lastStep.DetailedString(), loadedLastBlueprint.Steps[i].DetailedString())
	}

	reLoadedLastBlueprint, err := LoadOrCreateAndSaveLastBlueprintWithValidation(
		ctx,
		host,
		store,
		targetBlueprint,
	)
	require.NoError(t, err)
	require.Equal(t, len(lastBlueprint.Steps), len(reLoadedLastBlueprint.Steps))
	for i, lastStep := range lastBlueprint.Steps {
		require.Equal(t, lastStep.DetailedString(), reLoadedLastBlueprint.Steps[i].DetailedString())
	}

	err = host.Remove(ctx, filePath)
	require.NoError(t, err)

	_, err = LoadOrCreateAndSaveLastBlueprintWithValidation(
		ctx,
		host,
		store,
		targetBlueprint,
	)
	require.ErrorContains(t, err, "host state has changed")
}
