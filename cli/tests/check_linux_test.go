package tests

import (
	"testing"

	"github.com/fornellas/resonance/cli/tests/resources"
	"github.com/fornellas/resonance/resource"
)

func TestCheckNoPreviousState(t *testing.T) {
	stateRoot, _ := setupDirs(t)

	args := []string{
		"check",
		"--log-level=trace",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
	}

	runCommand(t, Cmd{
		Args:             args,
		ExpectedCode:     1,
		ExpectedInOutput: "No previously saved host state available to check",
	})
}

func TestCheckClean(t *testing.T) {
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

	t.Run("check is clean", func(t *testing.T) {

		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
		})
		args := []string{
			"check",
			"--log-level=trace",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "State is clean",
		})
	})
}

func TestCheckDirty(t *testing.T) {
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

	t.Run("check is dirty", func(t *testing.T) {

		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name: "foo",
				ReturnState: resources.IndividualState{
					Value: "fooDirty",
				},
			}},
		})
		args := []string{
			"check",
			"--log-level=trace",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		runCommand(t, Cmd{
			Args:             args,
			ExpectedCode:     1,
			ExpectedInOutput: "Previous host state is not clean:",
		})
	})
}
