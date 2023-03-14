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

func setupTestType(t *testing.T, testFuncCalls []resource.TestFuncCall) {
	resource.TestT = t
	resource.TestExpectedFuncCalls = testFuncCalls
	t.Cleanup(func() {
		if len(resource.TestExpectedFuncCalls) > 0 {
			t.Errorf("expected calls pending:\n%v", resource.TestExpectedFuncCalls)
		}
	})
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
	logged := false
	logCmd := func() {
		logged = true
		t.Logf("%s\n%s", cmd, outputBuffer.String())
	}
	t.Cleanup(func() {
		if !logged && t.Failed() {
			logCmd()
		}
	})

	type expectedExit struct{}

	cli.ExitFunc = func(code int) {
		t.Logf("ExitFunc(%d)", code)
		if cmd.ExpectedCode != code {
			logCmd()
			t.Fatalf("expected exit code %d: got %d", cmd.ExpectedCode, code)
		}
		panic(expectedExit{})
	}
	defer func() {
		switch p := recover(); p {
		case nil:
			if cmd.ExpectedCode != 0 {
				logCmd()
				t.Fatalf("expected exit code %d: got %d", cmd.ExpectedCode, 0)
			}
		case expectedExit{}:
			if cmd.ExpectedOutput != "" {
				if !strings.Contains(outputBuffer.String(), cmd.ExpectedOutput) {
					logCmd()
					t.Fatalf("output does not contain %#v", cmd.ExpectedOutput)
				}
			}
		default:
			panic(p)
		}
	}()
	command := cli.Cmd
	command.SetArgs(cmd.Args)
	command.SetOut(&outputBuffer)
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
}
