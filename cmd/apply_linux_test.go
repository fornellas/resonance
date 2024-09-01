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

func TestApply(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "store")
	resourcesDir := filepath.Join(tempDir, "resources")
	resourcesFile := filepath.Join(resourcesDir, "resources.yaml")
	filesDir := filepath.Join(tempDir, "files")
	err := os.MkdirAll(filesDir, fs.FileMode(0755))
	require.NoError(t, err)
	fileContent := "bar"

	resources := resouresPkg.Resources{
		&resouresPkg.File{
			Path:    filepath.Join(filesDir, "bar"),
			Perm:    os.FileMode(0644),
			Content: fileContent,
			Uid:     uint32(os.Geteuid()),
			Gid:     uint32(os.Getgid()),
		},
	}

	WriteResourcesFile(t, resourcesFile, resources)

	t.Run("first apply", func(t *testing.T) {
		(&TestCmd{
			Args: []string{
				"apply",
				"--target-localhost",
				"--store", "localhost",
				"--store-localhost-path", storeDir,
				resourcesDir,
			},
			ExpectStderrContains: []string{fmt.Sprintf(`⚙️ Applying
  File:🔧%s/bar
    diff:
      path: %s/bar
      -absent: true
      +content: bar
      +perm: 420
      +uid: 1000
      +gid: 1000
🧹 State cleanup`,
				filesDir, filesDir,
			)},
		}).Run(t)
	})

	t.Run("re-apply", func(t *testing.T) {
		(&TestCmd{
			Args: []string{
				"apply",
				"--target-localhost",
				"--store", "localhost",
				"--store-localhost-path", storeDir,
				resourcesDir,
			},
			ExpectStderrContains: []string{fmt.Sprintf(`⚙️ Applying
  File:✅%s/bar
🧹 State cleanup`,
				filesDir,
			)},
		}).Run(t)
	})
}
