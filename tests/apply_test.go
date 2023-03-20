package tests

import (
	"errors"
	"testing"

	"github.com/fornellas/resonance/resource"
)

func TestApplyNoYamlResourceFiles(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)
	runCommand(t, Cmd{
		Args: []string{
			"apply",
			"--log-level=debug",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
			resourcesRoot,
		},
		ExpectedCode:   1,
		ExpectedOutput: "no .yaml resource files found",
	})
}

func TestApplySuccess(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	args := []string{
		"apply",
		"--log-level=debug",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
		resourcesRoot,
	}

	fooState := TestState{
		Value: "foo",
	}

	barState := TestState{
		Value: "bar",
	}

	t.Run("First apply", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Test[foo]",
					State:    fooState,
				},
				{
					TypeName: "Test[bar]",
					State:    barState,
				},
			},
		})
		setupTestType(t, []TestFuncCall{
			// Loading resources
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
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
			{Apply: &TestFuncApply{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:           args,
			ExpectedOutput: "Success",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("Idempotency", func(t *testing.T) {
		setupTestType(t, []TestFuncCall{
			// Loading resources
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &TestFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:           args,
			ExpectedOutput: "Success",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("Destroy old resources", func(t *testing.T) {
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
			// Loading saved host state
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
			// Executing plan
			{Destroy: &TestFuncDestroy{
				Name: "bar",
			}},
		})
		runCommand(t, Cmd{
			Args:           args,
			ExpectedOutput: "Success",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("Idempotency", func(t *testing.T) {
		setupTestType(t, []TestFuncCall{
			// Loading resources
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
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
		runCommand(t, Cmd{
			Args:           args,
			ExpectedOutput: "Success",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("Apply new resource", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Test[foo]",
					State:    fooState,
				},
				{
					TypeName: "Test[bar]",
					State:    barState,
				},
			},
		})
		setupTestType(t, []TestFuncCall{
			// Loading resources
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &TestFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: nil,
			}},
			// Executing plan
			{Apply: &TestFuncApply{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:           args,
			ExpectedOutput: "Success",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("Idempotency", func(t *testing.T) {
		setupTestType(t, []TestFuncCall{
			// Loading resources
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &TestFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:           args,
			ExpectedOutput: "Success",
		})
	})
}

func TestApplyFailureWithSuccessfulRollback(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	args := []string{
		"apply",
		"--log-level=debug",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
		resourcesRoot,
	}

	fooState := TestState{
		Value: "foo",
	}

	barState := TestState{
		Value: "bar",
	}

	t.Run("First apply", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Test[foo]",
					State:    fooState,
				},
				{
					TypeName: "Test[bar]",
					State:    barState,
				},
			},
		})
		setupTestType(t, []TestFuncCall{
			// Loading resources
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
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
			{Apply: &TestFuncApply{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:           args,
			ExpectedOutput: "Success",
		})
	})

	if t.Failed() {
		return
	}

	fooNewState := TestState{
		Value: "fooNew",
	}

	barNewState := TestState{
		Value: "barNew",
	}

	t.Run("Apply failure with rollback", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Test[foo]",
					State:    fooNewState,
				},
				{
					TypeName: "Test[bar]",
					State:    barNewState,
				},
			},
		})
		setupTestType(t, []TestFuncCall{
			// Loading resources
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &TestFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
			// Executing plan
			{Apply: &TestFuncApply{
				Name:  "foo",
				State: fooNewState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooNewState,
			}},
			{Apply: &TestFuncApply{
				Name:        "bar",
				State:       barNewState,
				ReturnError: errors.New("barNew failed"),
			}},
			// Rollback: Reading host state
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooNewState,
			}},
			{GetState: &TestFuncGetState{
				Name: "bar",
				ReturnState: TestState{
					Value: "barBroken",
				},
			}},
			// Rollback: Executing plan
			{Apply: &TestFuncApply{
				Name:  "foo",
				State: fooState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{Apply: &TestFuncApply{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:           args,
			ExpectedCode:   1,
			ExpectedOutput: "Failed to apply, rollback to previously saved state successful",
		})
	})

}
