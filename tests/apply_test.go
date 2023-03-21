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

	fooState := resources.IndividualState{
		Value: "foo",
	}

	barState := resources.IndividualState{
		Value: "bar",
	}

	t.Run("First apply", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Individual[foo]",
					State:    fooState,
				},
				{
					TypeName: "Individual[bar]",
					State:    barState,
				},
			},
		})
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "bar",
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
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &resources.IndividualFuncGetState{
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
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &resources.IndividualFuncGetState{
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
					TypeName: "Individual[foo]",
					State:    fooState,
				},
			},
		})
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
			// Executing plan
			{Destroy: &resources.IndividualFuncDestroy{
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
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
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
					TypeName: "Individual[foo]",
					State:    fooState,
				},
				{
					TypeName: "Individual[bar]",
					State:    barState,
				},
			},
		})
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "bar",
				ReturnState: nil,
			}},
			// Executing plan
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &resources.IndividualFuncGetState{
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
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &resources.IndividualFuncGetState{
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

	fooState := resources.IndividualState{
		Value: "foo",
	}

	barState := resources.IndividualState{
		Value: "bar",
	}

	fooNewState := resources.IndividualState{
		Value: "fooNew",
	}

	barNewState := resources.IndividualState{
		Value: "barNew",
	}

	t.Run("apply", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Individual[foo]",
					State:    fooState,
				},
				{
					TypeName: "Individual[bar]",
					State:    barState,
				},
			},
		})
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "bar",
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
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &resources.IndividualFuncGetState{
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
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooNewState,
			}},
			{GetState: &resources.IndividualFuncGetState{
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

func TestApplyFailureWithRollback(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	args := []string{
		"apply",
		"--log-level=debug",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
		resourcesRoot,
	}

	fooState := resources.IndividualState{
		Value: "foo",
	}

	barState := resources.IndividualState{
		Value: "bar",
	}

	t.Run("First apply", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Individual[foo]",
					State:    fooState,
				},
				{
					TypeName: "Individual[bar]",
					State:    barState,
				},
			},
		})
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "bar",
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
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &resources.IndividualFuncGetState{
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

	fooNewState := resources.IndividualState{
		Value: "fooNew",
	}

	barNewState := resources.IndividualState{
		Value: "barNew",
	}

	t.Run("Apply failure with rollback", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Individual[foo]",
					State:    fooNewState,
				},
				{
					TypeName: "Individual[bar]",
					State:    barNewState,
				},
			},
		})
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
			// Executing plan
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "foo",
				State: fooNewState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooNewState,
			}},
			{Configure: &resources.IndividualFuncConfigure{
				Name:        "bar",
				State:       barNewState,
				ReturnError: errors.New("barNew failed"),
			}},
			// Rollback: Reading host state
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooNewState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name: "bar",
				ReturnState: resources.IndividualState{
					Value: "barBroken",
				},
			}},
			// Rollback: Executing plan
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "foo",
				State: fooState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "bar",
				State: barState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "bar",
				ReturnState: barState,
			}},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedCode:     1,
			ExpectedInOutput: "Failed, rollback to previously saved state successful",
		})
	})
}
