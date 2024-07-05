package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/internal/resource"
)

func CreateResourceYamls(t *testing.T, resourcesRoot string, resourcesMap map[string]resource.Resources) {
	require.NoError(t, os.Mkdir(resourcesRoot, os.FileMode(0700)))
	for name, resources := range resourcesMap {
		bundleBytes, err := yaml.Marshal(resources)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(resourcesRoot, name), bundleBytes, os.FileMode(0600),
		); err != nil {
			t.Fatal(err)
		}
	}
}

type TestCmd struct {
	Args         []string
	ExpectedCode int
}

func (c TestCmd) String() string {
	return strings.Join(append([]string{"resonance"}, c.Args...), " ")
}

func (c *TestCmd) Run(t *testing.T) {
	Exit = func(code int) {
		if c.ExpectedCode != code {
			t.Fatalf("%v exited %d, expected %d", c, code, c.ExpectedCode)
		}
		runtime.Goexit()
	}

	RootCmd.SetArgs(c.Args)

	Reset()

	if err := RootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
}
