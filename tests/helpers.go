package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/cli"
	"github.com/fornellas/resonance/resource"
)

func setupTestType(t *testing.T, testFuncCalls []resource.TestFuncCall) func() {
	resource.TestT = t
	resource.TestExpectedFuncCalls = testFuncCalls
	return func() {
		if len(resource.TestExpectedFuncCalls) > 0 {
			t.Fatalf("expected calls pending: %v", resource.TestExpectedFuncCalls)
		}
	}
}

func setupDirs(t *testing.T) (string, string) {
	prefix := t.TempDir()

	stateRoot := filepath.Join(prefix, "state")
	if err := os.Mkdir(stateRoot, os.FileMode(0700)); err != nil {
		t.Fatal(err)
	}

	resourcesRoot := filepath.Join(prefix, "resources")
	if err := os.Mkdir(resourcesRoot, os.FileMode(0700)); err != nil {
		t.Fatal(err)
	}

	return stateRoot, resourcesRoot
}

func setupBundles(t *testing.T, resourcesRoot string, bundleMap map[string]resource.Bundle) {
	for name, bundle := range bundleMap {
		bundleBytes, err := yaml.Marshal(bundle)
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

type Cmd struct {
	Args           []string
	ExpectedCode   int
	ExpectedOutput string
}

func (c Cmd) String() string {
	return strings.Join(c.Args, " ")
}

func runCommand(t *testing.T, cmd Cmd) {
	var outputBuffer bytes.Buffer
	cli.ExitFunc = func(code int) {
		if cmd.ExpectedCode != code {
			t.Logf("%s\n%s", cmd, outputBuffer.String())
			t.Fatalf("expected exit code %d: got %d", cmd.ExpectedCode, code)
		}
	}
	command := cli.Cmd
	command.SetArgs(cmd.Args)
	command.SetOut(&outputBuffer)
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}

	if cmd.ExpectedCode != 0 {
		t.Logf("%s\n%s", cmd, outputBuffer.String())
		t.Fatalf("expected exit code %d: got %d", cmd.ExpectedCode, 0)
	}
	if cmd.ExpectedOutput != "" {
		if !strings.Contains(outputBuffer.String(), cmd.ExpectedOutput) {
			t.Logf("%s\n%s", cmd, outputBuffer.String())
			t.Fatalf("output does not contain %#v", cmd.ExpectedOutput)
		}
	}
}
