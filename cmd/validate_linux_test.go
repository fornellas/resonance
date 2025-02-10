package main

import (
	"path/filepath"
	"testing"

	"github.com/fornellas/resonance/host/types"
	resouresPkg "github.com/fornellas/resonance/resources"
)

func TestValidate(t *testing.T) {
	tempDir := t.TempDir()
	resourcesDir := tempDir
	resourcesFile := filepath.Join(tempDir, "resources.yaml")

	var mode types.FileMode = 0644

	regularFile := "foo"
	successResources := resouresPkg.Resources{
		&resouresPkg.File{
			Path:        filepath.Join(tempDir, "foo"),
			RegularFile: &regularFile,
			Mode:        &mode,
		},
	}

	t.Run("directory", func(t *testing.T) {
		WriteResourcesFile(t, resourcesFile, successResources)
		cmd := TestCmd{
			Args: []string{
				"validate",
				"--target-localhost",
				resourcesDir,
			},
			ExpectStderrContains: []string{
				resourcesFile,
				"Validation successful",
			},
		}
		cmd.Run(t)
	})

	type testCase struct {
		Name                 string
		Resources            resouresPkg.Resources
		ExpectedCode         int
		ExpectStderrContains []string
	}

	badUser := "bad-user-name"
	for _, tc := range []testCase{
		{
			Name:                 "success",
			Resources:            successResources,
			ExpectStderrContains: []string{"Validation successful"},
		},
		{
			Name: "fail update",
			Resources: resouresPkg.Resources{
				&resouresPkg.File{
					Path: filepath.Join(tempDir, "foo"),
					Mode: &mode,
					User: &badUser,
				},
			},
			ExpectedCode:         1,
			ExpectStderrContains: []string{"user: unknown user bad-user-name"},
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			WriteResourcesFile(t, resourcesFile, tc.Resources)
			cmd := TestCmd{
				Args: []string{
					"validate",
					"--target-localhost",
					resourcesFile,
				},
				ExpectedCode:         tc.ExpectedCode,
				ExpectStderrContains: tc.ExpectStderrContains,
			}
			cmd.Run(t)
		})
	}
}
