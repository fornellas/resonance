package tests

import (
	"testing"

	"github.com/fornellas/resonance/resource"
)

func TestApplyNoYamlResourceFiles(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)
	runCommand(t, Cmd{
		Args:           []string{"apply", "--localhost", "--state-root", stateRoot, resourcesRoot},
		ExpectedCode:   1,
		ExpectedOutput: "no .yaml resource files found",
	})
}

func TestApplySuccess(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	fooDesiredState := resource.TestState{
		Value: "foo",
	}

	barDesiredState := resource.TestState{
		Value: "foo",
	}

	t.Run("First apply", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Bundle{
			"test.yaml": resource.Bundle{
				{
					TypeName: "Test[foo]",
					State:    fooDesiredState,
				},
				{
					TypeName: "Test[bar]",
					State:    barDesiredState,
				},
			},
		})
		setupTestType(t, []resource.TestFuncCall{
			// Load
			{ValidateName: &resource.TestFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resource.TestFuncValidateName{
				Name: "bar",
			}},
			// Plan
			{GetState: &resource.TestFuncGetState{
				Name:        "foo",
				ReturnState: nil,
			}},
			{GetState: &resource.TestFuncGetState{
				Name:        "bar",
				ReturnState: nil,
			}},
			{GetState: &resource.TestFuncGetState{ // FIXME should not do double call here
				Name:        "foo",
				ReturnState: nil,
			}},
			{GetState: &resource.TestFuncGetState{ // FIXME should not do double call here
				Name:        "bar",
				ReturnState: nil,
			}},
			// Apply
			{Apply: &resource.TestFuncApply{
				Name:  "foo",
				State: fooDesiredState,
			}},
			{GetState: &resource.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooDesiredState,
			}},
			{DiffStates: &resource.TestFuncDiffStates{
				DesiredState: fooDesiredState,
				CurrentState: fooDesiredState,
			}},
			{Apply: &resource.TestFuncApply{
				Name:  "bar",
				State: barDesiredState,
			}},
			{GetState: &resource.TestFuncGetState{
				Name:        "bar",
				ReturnState: barDesiredState,
			}},
			{DiffStates: &resource.TestFuncDiffStates{
				DesiredState: barDesiredState,
				CurrentState: barDesiredState,
			}},
			// TODO make this check optional
			{GetState: &resource.TestFuncGetState{
				Name:        "foo",
				ReturnState: fooDesiredState,
			}},
			{DiffStates: &resource.TestFuncDiffStates{
				DesiredState: fooDesiredState,
				CurrentState: fooDesiredState,
			}},
			{GetState: &resource.TestFuncGetState{
				Name:        "bar",
				ReturnState: barDesiredState,
			}},
			{DiffStates: &resource.TestFuncDiffStates{
				DesiredState: barDesiredState,
				CurrentState: barDesiredState,
			}},
		})
		runCommand(t, Cmd{
			Args:           []string{"apply", "--localhost", "--state-root", stateRoot, resourcesRoot},
			ExpectedOutput: "Success",
		})
	})

	// t.Run("Idempotent apply", func(t *testing.T) {})

	// t.Run("Destroy old resources", func(t *testing.T) {})

	// t.Run("Idempotent apply", func(t *testing.T) {})

	// t.Run("Apply new resource", func(t *testing.T) {})

	// t.Run("Idempotent apply", func(t *testing.T) {})
}

// func TestApplyFailureWithRollback(t *testing.T) {

// 	t.Run("First apply", func(t *testing.T) {})

// 	t.Run("Apply with failure", func(t *testing.T) {})

// 	t.Run("Idempotent apply", func(t *testing.T) {})
// }
