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

func setupTestInstance(t *testing.T, testFuncCalls []resource.TestFuncCall) func() {
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

func TestApplyNoYamlResourceFiles(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)
	runCommand(t, Cmd{
		Args:           []string{"apply", "--localhost", "--state-root", stateRoot, resourcesRoot},
		ExpectedCode:   1,
		ExpectedOutput: "no .yaml resource files found",
	})
}

func TestApplySimple(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	fooDesiredState := resource.TestState{
		Value: "foo",
	}

	barDesiredState := resource.TestState{
		Value: "foo",
	}

	setupBundles(t, resourcesRoot, map[string]resource.Bundle{
		"test.yaml": resource.Bundle{
			{
				TypeName: "Test[foo]",
				State:    fooDesiredState,
			},
			{
				TypeName: "Test[bar]",
				State:    barDesiredState,
			},
		},
	})

	assertion := setupTestInstance(t, []resource.TestFuncCall{
		// Planning
		{ValidateName: &resource.TestFuncValidateName{
			Name: "foo",
		}},
		{ValidateName: &resource.TestFuncValidateName{
			Name: "bar",
		}},
		{GetState: &resource.TestFuncGetState{
			Name:        "foo",
			ReturnState: nil,
		}},
		{GetState: &resource.TestFuncGetState{
			Name:        "bar",
			ReturnState: nil,
		}},
		{GetState: &resource.TestFuncGetState{ // FIXME should not do double call here
			Name:        "foo",
			ReturnState: nil,
		}},
		{GetState: &resource.TestFuncGetState{ // FIXME should not do double call here
			Name:        "bar",
			ReturnState: nil,
		}},
		// Apply
		{Apply: &resource.TestFuncApply{
			Name:  "foo",
			State: &fooDesiredState,
		}},
		{GetState: &resource.TestFuncGetState{
			Name:        "foo",
			ReturnState: &fooDesiredState,
		}},
		{DiffStates: &resource.TestFuncDiffStates{
			DesiredState: &fooDesiredState,
			CurrentState: &fooDesiredState,
		}},
		{Apply: &resource.TestFuncApply{
			Name:  "bar",
			State: &barDesiredState,
		}},
		{GetState: &resource.TestFuncGetState{
			Name:        "bar",
			ReturnState: &barDesiredState,
		}},
		{DiffStates: &resource.TestFuncDiffStates{
			DesiredState: &barDesiredState,
			CurrentState: &barDesiredState,
		}},
		// TODO make this check optional
		{GetState: &resource.TestFuncGetState{
			Name:        "foo",
			ReturnState: &fooDesiredState,
		}},
		{DiffStates: &resource.TestFuncDiffStates{
			DesiredState: &fooDesiredState,
			CurrentState: &fooDesiredState,
		}},
		{GetState: &resource.TestFuncGetState{
			Name:        "bar",
			ReturnState: &barDesiredState,
		}},
		{DiffStates: &resource.TestFuncDiffStates{
			DesiredState: &barDesiredState,
			CurrentState: &barDesiredState,
		}},
	})

	runCommand(t, Cmd{
		Args:           []string{"apply", "--localhost", "--state-root", stateRoot, resourcesRoot},
		ExpectedCode:   1,
		ExpectedOutput: "no .yaml resource files found",
	})

	assertion()
}
