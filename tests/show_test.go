package tests

import (
	"testing"

	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/tests/resources"
)

func TestShowNoPreviousState(t *testing.T) {
	stateRoot, _ := setupDirs(t)

	args := []string{
		"show",
		"--log-level=debug",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
	}

	runCommand(t, Cmd{
		Args:             args,
		ExpectedCode:     1,
		ExpectedInOutput: "No previously saved host state available to show",
	})
}

func TestShow(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	fooState := resources.TestState{
		Value: "foo",
	}

	t.Run("apply", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Test[foo]",
					State:    fooState,
				},
			},
		})
		setupTestType(t, []resources.TestFuncCall{
			// Loading resources
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			// Executing plan
			{Configure: &resources.TestFuncConfigure{
				Name:  "foo",
				State: fooState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
		})
		args := []string{
			"apply",
			"--log-level=debug",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
			resourcesRoot,
		}
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Apply successful",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("show", func(t *testing.T) {

		setupTestType(t, []resources.TestFuncCall{
			// Loading saved host state
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
		})
		args := []string{
			"show",
			"--log-level=debug",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		runCommand(t, Cmd{
			Args: args,
			ExpectedInStdout: ("previous_bundle:\n" +
				"    - - resource: Test[foo]\n" +
				"        state:\n" +
				"            value: foo\n" +
				"        destroy: false\n"),
		})
	})
}
