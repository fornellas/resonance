package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	resouresPkg "github.com/fornellas/resonance/resources"
)

func TestPlan(t *testing.T) {
	t.Run("changes", func(t *testing.T) {
		tempDir := t.TempDir()
		storeDir := filepath.Join(tempDir, "store")
		resourcesDir := filepath.Join(tempDir, "resources")
		resourcesFile := filepath.Join(resourcesDir, "resources.yaml")
		filesDir := filepath.Join(tempDir, "files")

		resources := resouresPkg.Resources{
			&resouresPkg.File{
				Path:        filepath.Join(filesDir, "bar"),
				Mode:        0644,
				RegularFile: "bar",
				User:        "root",
				Group:       "root",
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
      +mode: 420
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
		var fileMode uint32 = 0644
		fileContent := "bar"
		err = os.WriteFile(filePath, []byte(fileContent), os.FileMode(fileMode))
		require.NoError(t, err)

		resources := resouresPkg.Resources{
			&resouresPkg.File{
				Path:        filePath,
				Mode:        fileMode,
				RegularFile: fileContent,
				Uid:         uint32(os.Geteuid()),
				Gid:         uint32(os.Getegid()),
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
