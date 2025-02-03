package plan

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	blueprintPkg "github.com/fornellas/resonance/blueprint"
	hostPkg "github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"

	"github.com/fornellas/resonance/log"

	resouresPkg "github.com/fornellas/resonance/resources"
	storePkg "github.com/fornellas/resonance/store"
)

func TestCreateAndStoreTargetBlueprint(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	host := hostPkg.Local{}

	regularFile := "foo"
	user := "root"

	targetResources := resouresPkg.Resources{
		&resouresPkg.File{
			Path:        "/bin",
			RegularFile: &regularFile,
			User:        &user,
		},
		&resouresPkg.File{
			Path:        "/lib",
			RegularFile: &regularFile,
			User:        &user,
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
			Path:        "/bin",
			RegularFile: &regularFile,
			Uid:         new(uint32),
			Gid:         new(uint32),
		},
		&resouresPkg.File{
			Path:        "/lib",
			RegularFile: &regularFile,
			Uid:         new(uint32),
			Gid:         new(uint32),
		},
	}, targetBlueprint.Resources())
}

func TestSaveOriginalResourcesState(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	host := hostPkg.Local{}

	storePath := filepath.Join(t.TempDir(), "store")
	store := storePkg.NewHostStore(host, storePath)

	filePath := filepath.Join(t.TempDir(), "foo")
	fileContent := "foo"
	var fileMode types.FileMode = 0644
	err := host.WriteFile(ctx, filePath, bytes.NewReader([]byte("foo")), fileMode)
	require.NoError(t, err)
	fileResource := &resouresPkg.File{
		Path:        filePath,
		RegularFile: &fileContent,
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
	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())
	require.Equal(t,
		&resouresPkg.File{
			Path:        filePath,
			RegularFile: &fileContent,
			Mode:        &fileMode,
			Uid:         &uid,
			Gid:         &gid,
		},
		resource,
	)
}

func TestLoadOrCreateAndSaveLastBlueprintWithValidation(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	host := hostPkg.Local{}

	storePath := filepath.Join(t.TempDir(), "store")
	store := storePkg.NewHostStore(host, storePath)

	filePath := filepath.Join(t.TempDir(), "foo")
	fileContent := "foo"
	var fileMode types.FileMode = 0644
	err := host.WriteFile(ctx, filePath, bytes.NewReader([]byte("foo")), fileMode)
	require.NoError(t, err)
	fileResource := &resouresPkg.File{
		Path:        filePath,
		RegularFile: &fileContent,
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
