package tests

import (
	"testing"

	"github.com/fornellas/resonance/cli/tests/resources"
	"github.com/fornellas/resonance/resource"
)

func TestShowNoPreviousState(t *testing.T) {
	stateRoot, _ := setupDirs(t)

	args := []string{
		"show",
		"--log-level=trace",
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

	fooState := resources.IndividualState{
		Value: "foo",
	}

	t.Run("apply", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Individual[foo]",
					State:    fooState,
				},
			},
		})
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			// Executing plan
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "foo",
				State: fooState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
		})
		args := []string{
			"apply",
			"--log-level=trace",
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

		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
		})
		args := []string{
			"show",
			"--log-level=trace",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		runCommand(t, Cmd{
			Args: args,
			ExpectedInStdout: ("previous_bundle:\n" +
				"    - - resource: Individual[foo]\n" +
				"        state:\n" +
				"            value: foo\n" +
				"        destroy: false\n"),
		})
	})
}
