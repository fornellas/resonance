package tests

import (
	"testing"

	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/tests/resources"
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

	fooState := resources.TestState{
		Value: "foo",
	}

	fooStateNew := resources.TestState{
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

	t.Run("refresh", func(t *testing.T) {
		setupTestType(t, []resources.TestFuncCall{
			// Loading saved host state
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
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
		setupTestType(t, []resources.TestFuncCall{
			// Loading resources
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			// Loading saved host state
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			// Reading host state
			{GetState: &resources.TestFuncGetState{
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
			ExpectedInOutput: "Apply successful",
		})
	})
}
