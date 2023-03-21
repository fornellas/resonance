package tests

import (
	"errors"
	"testing"

	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/tests/resources"
)

func TestRollbackNoPreviousState(t *testing.T) {
	stateRoot, _ := setupDirs(t)

	args := []string{
		"rollback",
		"--log-level=trace",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
	}

	runCommand(t, Cmd{
		Args:             args,
		ExpectedCode:     1,
		ExpectedInOutput: "No previously saved host state available to rollback from",
	})
}

func TestRollbackNoPreviousRollbackState(t *testing.T) {
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
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
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

	t.Run("rollback not required", func(t *testing.T) {
		args := []string{
			"rollback",
			"--log-level=trace",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedCode:     1,
			ExpectedInOutput: "No rollback required for saved host state",
		})
	})
}

func TestRollback(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	fooState := resources.IndividualState{
		Value: "foo",
	}

	fooStateNew := resources.IndividualState{
		Value: "fooNew",
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
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
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

	t.Run("apply broken rollback", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Individual[foo]",
					State:    fooStateNew,
				},
			},
		})
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Loading saved state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooState,
			}},
			// Executing plan
			{Configure: &resources.IndividualFuncConfigure{
				Name:        "foo",
				State:       fooStateNew,
				ReturnError: errors.New("fooFail"),
			}},
			// Rollback: Reading host state
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooStateBroken,
			}},
			// Rollback: Executing plan
			{Configure: &resources.IndividualFuncConfigure{
				Name:        "foo",
				State:       fooState,
				ReturnError: errors.New("fooFailAgain"),
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
			ExpectedCode:     1,
			ExpectedInOutput: "Rollback failed",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("rollback", func(t *testing.T) {
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
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
			"rollback",
			"--log-level=trace",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
		}
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Rollback successful",
		})
	})

	if t.Failed() {
		return
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
		resources.SetupIndividualType(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			// Loading resources
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
