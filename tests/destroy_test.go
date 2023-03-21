package tests

import (
	"errors"
	"testing"

	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/tests/resources"
)

func TestDestroy(t *testing.T) {
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
			ExpectedInOutput: "Success",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("destroy", func(t *testing.T) {
		setupTestType(t, []resources.TestFuncCall{
			// Loading saved host state
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			// Executing plan
			{Destroy: &resources.TestFuncDestroy{
				Name: "foo",
			}},
		})
		args := []string{
			"destroy",
			"--log-level=debug",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Success",
		})
	})
}

func TestDestroyFailureWithSuccessfulRollback(t *testing.T) {
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
			ExpectedInOutput: "Success",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("destroy with rollback", func(t *testing.T) {
		setupTestType(t, []resources.TestFuncCall{
			// Loading saved host state
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			// Executing plan
			{Destroy: &resources.TestFuncDestroy{
				Name:        "foo",
				ReturnError: errors.New("fooError"),
			}},
			// Rollback: Reading host state
			{GetState: &resources.TestFuncGetState{
				Name: "foo",
				ReturnState: resources.TestState{
					Value: "fooError",
				},
			}},
			// Rollback: Applying changes
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
			"destroy",
			"--log-level=debug",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		runCommand(t, Cmd{
			Args:             args,
			ExpectedCode:     1,
			ExpectedInOutput: "Failed to apply, rollback to previously saved state successful",
		})
	})
}
