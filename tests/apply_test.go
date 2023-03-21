package tests

import (
	"errors"
	"testing"

	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/tests/resources"
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
		ExpectedCode:     1,
		ExpectedInOutput: "no .yaml resource files found",
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

	fooState := resources.TestState{
		Value: "foo",
	}

	barState := resources.TestState{
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
		setupTestType(t, []resources.TestFuncCall{
			// Loading resources
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
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
			{Configure: &resources.TestFuncConfigure{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Apply successful",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("Idempotency", func(t *testing.T) {
		setupTestType(t, []resources.TestFuncCall{
			// Loading resources
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Apply successful",
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
		setupTestType(t, []resources.TestFuncCall{
			// Loading resources
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			// Loading saved host state
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
			// Executing plan
			{Destroy: &resources.TestFuncDestroy{
				Name: "bar",
			}},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Apply successful",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("Idempotency", func(t *testing.T) {
		setupTestType(t, []resources.TestFuncCall{
			// Loading resources
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			// Loading saved host state
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Apply successful",
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
		setupTestType(t, []resources.TestFuncCall{
			// Loading resources
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
				ReturnState: nil,
			}},
			// Executing plan
			{Configure: &resources.TestFuncConfigure{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Apply successful",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("Idempotency", func(t *testing.T) {
		setupTestType(t, []resources.TestFuncCall{
			// Loading resources
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Apply successful",
		})
	})
}

func TestApplyDirtyState(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	args := []string{
		"apply",
		"--log-level=debug",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
		resourcesRoot,
	}

	fooState := resources.TestState{
		Value: "foo",
	}

	barState := resources.TestState{
		Value: "bar",
	}

	fooNewState := resources.TestState{
		Value: "fooNew",
	}

	barNewState := resources.TestState{
		Value: "barNew",
	}

	t.Run("apply", func(t *testing.T) {
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
		setupTestType(t, []resources.TestFuncCall{
			// Loading resources
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
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
			{Configure: &resources.TestFuncConfigure{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Apply successful",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("apply with dirty state", func(t *testing.T) {
		setupTestType(t, []resources.TestFuncCall{
			// Loading resources
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooNewState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
				ReturnState: barNewState,
			}},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedCode:     1,
			ExpectedInOutput: "Host state is not clean",
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

	fooState := resources.TestState{
		Value: "foo",
	}

	barState := resources.TestState{
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
		setupTestType(t, []resources.TestFuncCall{
			// Loading resources
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
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
			{Configure: &resources.TestFuncConfigure{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Apply successful",
		})
	})

	if t.Failed() {
		return
	}

	fooNewState := resources.TestState{
		Value: "fooNew",
	}

	barNewState := resources.TestState{
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
		setupTestType(t, []resources.TestFuncCall{
			// Loading resources
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.TestFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
			// Executing plan
			{Configure: &resources.TestFuncConfigure{
				Name:  "foo",
				State: fooNewState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooNewState,
			}},
			{Configure: &resources.TestFuncConfigure{
				Name:        "bar",
				State:       barNewState,
				ReturnError: errors.New("barNew failed"),
			}},
			// Rollback: Reading host state
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooNewState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name: "bar",
				ReturnState: resources.TestState{
					Value: "barBroken",
				},
			}},
			// Rollback: Executing plan
			{Configure: &resources.TestFuncConfigure{
				Name:  "foo",
				State: fooState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{Configure: &resources.TestFuncConfigure{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &resources.TestFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedCode:     1,
			ExpectedInOutput: "Failed to apply, rollback to previously saved state successful",
		})
	})
}
