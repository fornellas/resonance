package tests

import (
	"testing"

	"github.com/fornellas/resonance/resource"
)

func TestCheckNoPreviousState(t *testing.T) {
	stateRoot, _ := setupDirs(t)

	args := []string{
		"check",
		"--log-level=debug",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
	}

	runCommand(t, Cmd{
		Args:           args,
		ExpectedCode:   1,
		ExpectedOutput: "No previously saved host state available to check",
	})
}

func TestCheckClean(t *testing.T) {
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
			{Apply: &TestFuncApply{
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

	t.Run("check is clean", func(t *testing.T) {

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
		})
		args := []string{
			"check",
			"--log-level=debug",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		runCommand(t, Cmd{
			Args:           args,
			ExpectedOutput: "State is clean",
		})
	})
}

func TestCheckDirty(t *testing.T) {
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
			{Apply: &TestFuncApply{
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

	t.Run("check is dirty", func(t *testing.T) {

		setupTestType(t, []TestFuncCall{
			// Loading saved host state
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name: "foo",
				ReturnState: TestState{
					Value: "fooDirty",
				},
			}},
		})
		args := []string{
			"check",
			"--log-level=debug",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		runCommand(t, Cmd{
			Args:           args,
			ExpectedCode:   1,
			ExpectedOutput: "Host state is not clean",
		})
	})
}
