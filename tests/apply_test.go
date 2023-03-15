package tests

import (
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"

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

	fooDesiredState := TestState{
		Value: "foo",
	}

	barDesiredState := TestState{
		Value: "bar",
	}

	t.Run("First apply", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
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
			{DiffStates: &TestFuncDiffStates{
				DesiredState: fooDesiredState,
				CurrentState: nil,
				ReturnDiffs: []diffmatchpatch.Diff{{
					Type: diffmatchpatch.DiffInsert,
					Text: "foo",
				}},
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: nil,
			}},
			{DiffStates: &TestFuncDiffStates{
				DesiredState: barDesiredState,
				CurrentState: nil,
				ReturnDiffs: []diffmatchpatch.Diff{{
					Type: diffmatchpatch.DiffInsert,
					Text: "bar",
				}},
			}},
			// Executing plan
			{Apply: &TestFuncApply{
				Name:  "foo",
				State: fooDesiredState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "foo",
				ReturnState: fooDesiredState,
			}},
			{DiffStates: &TestFuncDiffStates{
				DesiredState: fooDesiredState,
				CurrentState: fooDesiredState,
			}},
			{Apply: &TestFuncApply{
				Name:  "bar",
				State: barDesiredState,
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: barDesiredState,
			}},
			{DiffStates: &TestFuncDiffStates{
				DesiredState: barDesiredState,
				CurrentState: barDesiredState,
			}},
		})
		runCommand(t, Cmd{
			Args: []string{"apply",
				"--log-level=debug",
				"--force-color",
				"--localhost",
				"--state-root", stateRoot,
				resourcesRoot,
			},
			ExpectedOutput: "Success",
		})
	})

	if t.Failed() {
		return
	}

	t.Run("Idempotent apply", func(t *testing.T) {
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
				ReturnState: fooDesiredState,
			}},
			{DiffStates: &TestFuncDiffStates{
				DesiredState: fooDesiredState,
				CurrentState: fooDesiredState,
				ReturnDiffs: []diffmatchpatch.Diff{{
					Type: diffmatchpatch.DiffEqual,
					Text: "foo",
				}},
			}},
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: barDesiredState,
			}},
			{DiffStates: &TestFuncDiffStates{
				DesiredState: barDesiredState,
				CurrentState: barDesiredState,
				ReturnDiffs: []diffmatchpatch.Diff{{
					Type: diffmatchpatch.DiffEqual,
					Text: "bar",
				}},
			}},
		})
		runCommand(t, Cmd{
			Args: []string{"apply",
				"--log-level=debug",
				"--force-color",
				"--localhost",
				"--state-root", stateRoot,
				resourcesRoot,
			},
			ExpectedOutput: "Success",
		})
	})

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
