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

	args := []string{
		"apply",
		"--log-level=debug",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
		resourcesRoot,
	}

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
			{GetState: &TestFuncGetState{
				Name:        "bar",
				ReturnState: nil,
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
				ReturnDiffs: []diffmatchpatch.Diff{{
					Type: diffmatchpatch.DiffEqual,
					Text: "foo",
				}},
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
				ReturnDiffs: []diffmatchpatch.Diff{{
					Type: diffmatchpatch.DiffEqual,
					Text: "bar",
				}},
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
					State:    fooDesiredState,
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
			// Loading saved host state
			{ValidateName: &TestFuncValidateName{
				Name: "foo",
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
				ReturnState: nil,
			}},
			// Executing plan
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
				ReturnDiffs: []diffmatchpatch.Diff{{
					Type: diffmatchpatch.DiffEqual,
					Text: "bar",
				}},
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
			Args:           args,
			ExpectedOutput: "Success",
		})
	})

}

// func TestApplyFailureWithRollback(t *testing.T) {

// 	t.Run("First apply", func(t *testing.T) {})

// 	t.Run("Apply with failure", func(t *testing.T) {})

// 	t.Run("Idempotent apply", func(t *testing.T) {})
// }
