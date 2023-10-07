package tests

import (
	"errors"
	"testing"

	"github.com/fornellas/resonance/cli/tests/resources"
	"github.com/fornellas/resonance/resource"
)

func TestRestoreNoPreviousState(t *testing.T) {
	stateRoot, _ := setupDirs(t)

	args := []string{
		"restore",
		"--log-level=trace",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
	}

	runCommand(t, Cmd{
		Args:             args,
		ExpectedCode:     1,
		ExpectedInOutput: "No previously saved host state available to restore from",
	})
}

func TestRestore(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	fooState := resources.IndividualState{
		Value: "foo",
	}

	fooStateBroken := resources.IndividualState{
		Value: "fooBroken",
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

	t.Run("restore", func(t *testing.T) {
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooStateBroken,
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
			"restore",
			"--log-level=trace",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Restore successful",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("apply", func(t *testing.T) {
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Reading host state
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
}

func TestRestoreFailureWithRollback(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	fooState := resources.IndividualState{
		Value: "foo",
	}

	fooStateBroken := resources.IndividualState{
		Value: "fooBroken",
	}

	fooStateBad := resources.IndividualState{
		Value: "fooBad",
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

	t.Run("restore", func(t *testing.T) {
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooStateBroken,
			}},
			// Executing plan
			{Configure: &resources.IndividualFuncConfigure{
				Name:        "foo",
				State:       fooState,
				ReturnError: errors.New("fooFailed"),
			}},
			// Rollback: Reading host state
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooStateBad,
			}},
			// Rollback: Executing plan
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "foo",
				State: fooStateBroken,
			}},
			// Rollback: Reading host state
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooStateBroken,
			}},
		})
		args := []string{
			"restore",
			"--log-level=trace",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		runCommand(t, Cmd{
			Args:             args,
			ExpectedCode:     1,
			ExpectedInOutput: "Failed, rollback to previously saved state successful.",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("apply", func(t *testing.T) {
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Reading host state
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
}
