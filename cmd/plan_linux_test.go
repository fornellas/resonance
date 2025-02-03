package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host/types"
	resouresPkg "github.com/fornellas/resonance/resources"
)

func TestPlan(t *testing.T) {
	t.Run("changes", func(t *testing.T) {
		tempDir := t.TempDir()
		storeDir := filepath.Join(tempDir, "store")
		resourcesDir := filepath.Join(tempDir, "resources")
		resourcesFile := filepath.Join(resourcesDir, "resources.yaml")
		filesDir := filepath.Join(tempDir, "files")
		fileContent := "bar"
		user := "root"
		group := "root"
		var mode types.FileMode = 0644
		resources := resouresPkg.Resources{
			&resouresPkg.File{
				Path:        filepath.Join(filesDir, "bar"),
				Mode:        &mode,
				RegularFile: &fileContent,
				User:        &user,
				Group:       &group,
			},
		}

		WriteResourcesFile(t, resourcesFile, resources)

		cmd := TestCmd{
			Args: []string{
				"plan",
				"--target-localhost",
				"--store", "localhost",
				"--store-localhost-path", storeDir,
				resourcesDir,
			},
			ExpectStderrContains: []string{fmt.Sprintf(
				`💡 Actions
  File:🔧%s/bar
    diff:
      path: %s/bar
      -absent: true
      +regular_file: bar
      +mode: "0644"
      +uid: 0
      +gid: 0
🎆 Planning successful`,
				filesDir, filesDir,
			)},
		}
		cmd.Run(t)
	})

	t.Run("no changes", func(t *testing.T) {
		tempDir := t.TempDir()
		storeDir := filepath.Join(tempDir, "store")
		resourcesDir := filepath.Join(tempDir, "resources")
		resourcesFile := filepath.Join(resourcesDir, "resources.yaml")
		filesDir := filepath.Join(tempDir, "files")
		err := os.MkdirAll(filesDir, fs.FileMode(0755))
		require.NoError(t, err)
		filePath := filepath.Join(filesDir, "bar")
		var fileMode types.FileMode = 0644
		fileContent := "bar"
		err = os.WriteFile(filePath, []byte(fileContent), os.FileMode(fileMode))
		require.NoError(t, err)
		uid := uint32(os.Geteuid())
		gid := uint32(os.Getegid())

		resources := resouresPkg.Resources{
			&resouresPkg.File{
				Path:        filePath,
				Mode:        &fileMode,
				RegularFile: &fileContent,
				Uid:         &uid,
				Gid:         &gid,
			},
		}

		WriteResourcesFile(t, resourcesFile, resources)

		cmd := TestCmd{
			Args: []string{
				"plan",
				"--target-localhost",
				"--store", "localhost",
				"--store-localhost-path", storeDir,
				resourcesDir,
			},
			ExpectStderrContains: []string{fmt.Sprintf(
				`💡 Actions
  File:✅%s/bar
🎆 Planning successful`,
				filesDir,
			)},
		}
		cmd.Run(t)
	})
}
