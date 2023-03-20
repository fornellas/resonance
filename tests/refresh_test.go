package tests

import (
	"testing"

	"github.com/fornellas/resonance/resource"
)

func TestRefreshNoPreviousState(t *testing.T) {
	stateRoot, _ := setupDirs(t)

	args := []string{
		"refresh",
		"--log-level=debug",
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

func TestRefresh(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	fooState := TestState{
		Value: "foo",
	}

	fooStateNew := TestState{
		Value: "fooNew",
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
			Args:             args,
			ExpectedInOutput: "Success",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("refresh", func(t *testing.T) {

		setupTestType(t, []TestFuncCall{
			// Loading saved host state
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooStateNew,
			}},
		})
		args := []string{
			"refresh",
			"--log-level=debug",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Refresh successful",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("apply", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Test[foo]",
					State:    fooStateNew,
				},
			},
		})
		setupTestType(t, []TestFuncCall{
			// Loading resources
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			// Loading saved host state
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			// Reading host state
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooStateNew,
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
}
