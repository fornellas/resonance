package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	resouresPkg "github.com/fornellas/resonance/resources"
)

func TestPlan(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "state")
	resourcesDir := filepath.Join(tempDir, "resources")
	resourcesFile := filepath.Join(resourcesDir, "resources.yaml")
	filesDir := filepath.Join(tempDir, "files")

	resources := resouresPkg.Resources{
		&resouresPkg.File{
			Path:    filepath.Join(filesDir, "bar"),
			Perm:    os.FileMode(0644),
			Content: "bar",
			User:    "root",
		},
	}

	WriteResourcesFile(t, resourcesFile, resources)

	cmd := TestCmd{
		Args: []string{
			"plan",
			"--localhost",
			"--store", "localhost",
			"--store-localhost-path", storeDir,
			resourcesDir,
		},
		ExpectStderrContains: []string{fmt.Sprintf(
			`📝 Plan
  🛠️ File:%s/bar
    diff:
      +path: %s/bar
      +content: bar
      +perm: 420
🎆 Planning successful`,
			filesDir, filesDir,
		)},
	}
	cmd.Run(t)
}
