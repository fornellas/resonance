package main

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	resouresPkg "github.com/fornellas/resonance/resources"
)

func WriteResourcesFile(t *testing.T, path string, resources resouresPkg.Resources) {
	resourcesBytes, err := yaml.Marshal(resources)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, resourcesBytes, os.FileMode(0644)); err != nil {
		t.Fatal(err)
	}
}

func TestValidate(t *testing.T) {
	tempDir := t.TempDir()
	resourcesDir := tempDir
	resourcesFile := filepath.Join(tempDir, "resources.yaml")

	successResources := resouresPkg.Resources{
		&resouresPkg.File{
			Path: filepath.Join(tempDir, "foo"),
			Perm: os.FileMode(0644),
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
					Perm: os.FileMode(0644),
					User: "bad-user-name",
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
