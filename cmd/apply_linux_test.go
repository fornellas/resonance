package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host/types"
	resourcesPkg "github.com/fornellas/resonance/resources"
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
	fileUid := uint32(os.Geteuid())
	fileGid := uint32(os.Getgid())
	var mode types.FileMode = 0644
	resources := resourcesPkg.Resources{
		&resourcesPkg.File{
			Path:        filepath.Join(filesDir, "bar"),
			Mode:        &mode,
			RegularFile: &fileContent,
			Uid:         &fileUid,
			Gid:         &fileGid,
		},
	}

	WriteResourcesFile(t, resourcesFile, resources)

	t.Run("first apply", func(t *testing.T) {
		(&TestCmd{
			Args: []string{
				"apply",
				"--host-local",
				"--store", "local",
				"--store-local-path", storeDir,
				resourcesDir,
			},
			ExpectStderrContains: []string{fmt.Sprintf(`  ⚙️ Apply
    🚀 Action: Apply
      resources: File:🔧%s/bar
      diff:
        path: %s/bar
        -absent: true
        +regular_file: bar
        +mode: "0644"
        +uid: %d
        +gid: %d
      INFO Applying changes
  INFO 🎆 Apply successful
`,
				filesDir, filesDir, fileUid, fileGid,
			)},
		}).Run(t)
	})

	t.Run("re-apply", func(t *testing.T) {
		(&TestCmd{
			Args: []string{
				"apply",
				"--host-local",
				"--store", "local",
				"--store-local-path", storeDir,
				resourcesDir,
			},
			ExpectStderrContains: []string{fmt.Sprintf(`  ⚙️ Apply
    🚀 Action: Apply
      resources: File:✅%s/bar
      INFO Nothing to do
  INFO 🎆 Apply successful`,
				filesDir,
			)},
		}).Run(t)
	})
}
