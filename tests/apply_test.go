package test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fornellas/resonance/cli"
	"github.com/fornellas/resonance/resource"
)

func setupTestInstance(t *testing.T) {
	resource.TestInstance.T = t
	resource.TestInstance.ExpectedFuncCalls = []resource.TestFuncCall{}
	t.Cleanup(func() { resource.TestInstance.FinalAssert() })
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
		if cmd.ExpectedOutput != "" {
			if !strings.Contains(outputBuffer.String(), cmd.ExpectedOutput) {
				t.Logf("%s\n%s", cmd, outputBuffer.String())
				t.Fatalf("output does not contain %#v", cmd.ExpectedOutput)
			}
		}
	}
	command := cli.Cmd
	command.SetArgs(cmd.Args)
	command.SetOut(&outputBuffer)
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestApply(t *testing.T) {
	setupTestInstance(t)

	stateRoot, resourcesRoot := setupDirs(t)

	runCommand(t, Cmd{
		Args: []string{
			"apply",
			"--localhost",
			"--state-root", stateRoot,
			resourcesRoot,
		},
		ExpectedCode:   1,
		ExpectedOutput: "no .yaml resource files found",
	})
}
