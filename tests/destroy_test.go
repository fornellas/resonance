package tests

import (
	"errors"
	"testing"

	"github.com/fornellas/resonance/resource"
)

func TestDestroy(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	fooState := TestState{
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
		setupTestType(t, []TestFuncCall{
			// Loading resources
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			// Executing plan
			{Configure: &TestFuncConfigure{
				Name:  "foo",
				State: fooState,
			}},
			{GetState: &TestFuncGetState{
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
			Args:           args,
			ExpectedOutput: "Success",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("destroy", func(t *testing.T) {
		setupTestType(t, []TestFuncCall{
			// Loading saved host state
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			// Executing plan
			{Destroy: &TestFuncDestroy{
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
			Args:           args,
			ExpectedOutput: "Success",
		})
	})
}

func TestDestroyFailureWithSuccessfulRollback(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	fooState := TestState{
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
		setupTestType(t, []TestFuncCall{
			// Loading resources
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			// Executing plan
			{Configure: &TestFuncConfigure{
				Name:  "foo",
				State: fooState,
			}},
			{GetState: &TestFuncGetState{
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
			Args:           args,
			ExpectedOutput: "Success",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("destroy with rollback", func(t *testing.T) {
		setupTestType(t, []TestFuncCall{
			// Loading saved host state
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			// Executing plan
			{Destroy: &TestFuncDestroy{
				Name:        "foo",
				ReturnError: errors.New("fooError"),
			}},
			// Rollback: Reading host state
			{GetState: &TestFuncGetState{
				Name: "foo",
				ReturnState: TestState{
					Value: "fooError",
				},
			}},
			// Rollback: Applying changes
			{Configure: &TestFuncConfigure{
				Name:  "foo",
				State: fooState,
			}},
			{GetState: &TestFuncGetState{
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
			Args:           args,
			ExpectedCode:   1,
			ExpectedOutput: "Failed to apply, rollback to previously saved state successful",
		})
	})
}
