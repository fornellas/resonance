package tests

import (
	"errors"
	"testing"

	"github.com/fornellas/resonance/cli/tests/resources"
	"github.com/fornellas/resonance/resource"
)

func TestApplyNoYamlResourceFiles(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)
	runCommand(t, Cmd{
		Args: []string{
			"apply",
			"--log-level=trace",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
			resourcesRoot,
		},
		ExpectedCode:     1,
		ExpectedInOutput: "no .yaml resource files found",
	})
}

func TestApplyIndividual(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	args := []string{
		"apply",
		"--log-level=trace",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
		resourcesRoot,
	}

	fooState := resources.IndividualState{
		Value: "foo",
	}

	fooOriginalState := resources.IndividualState{
		Value: "fooOriginal",
	}

	barState := resources.IndividualState{
		Value: "bar",
	}

	bazState := resources.IndividualState{
		Value: "baz",
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
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
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
				ReturnState: fooOriginalState,
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
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
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

	t.Run("Old resources", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"test.yaml": resource.Resources{
				{
					TypeName: "Individual[baz]",
					State:    bazState,
				},
			},
		})
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "baz",
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
				Name:        "baz",
				ReturnState: nil,
			}},
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
				Name:  "baz",
				State: bazState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "baz",
				ReturnState: bazState,
			}},
			// {Configure: &resources.IndividualFuncConfigure{
			// 	Name:  "foo",
			// 	State: fooOriginalState,
			// }},
			// // TODO validate
			// {GetState: &resources.IndividualFuncGetState{
			// 	Name:        "foo",
			// 	ReturnState: fooOriginalState,
			// }},
			// {Configure: &resources.IndividualFuncConfigure{
			// 	Name:  "bar",
			// 	State: nil,
			// }},
			// {GetState: &resources.IndividualFuncGetState{
			// 	Name:        "bar",
			// 	ReturnState: nil,
			// }},
		})
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInOutput: "Apply successful",
		})
	})

	if t.Failed() {
		return
	}

	// t.Run("Idempotency", func(t *testing.T) {
	// 	resources.SetupIndividualType(t, []resources.IndividualFuncCall{
	// 		// Loading resources
	// 		{ValidateName: &resources.IndividualFuncValidateName{
	// 			Name: "foo",
	// 		}},
	// 		// Loading saved host state
	// 		{ValidateName: &resources.IndividualFuncValidateName{
	// 			Name: "foo",
	// 		}},
	// 		// Reading Host State
	// 		{GetState: &resources.IndividualFuncGetState{
	// 			Name:        "foo",
	// 			ReturnState: fooState,
	// 		}},
	// 	})
	// 	runCommand(t, Cmd{
	// 		Args:             args,
	// 		ExpectedInOutput: "Apply successful",
	// 	})
	// })

	// if t.Failed() {
	// 	return
	// }

	// t.Run("Apply new resource", func(t *testing.T) {
	// 	setupBundles(t, resourcesRoot, map[string]resource.Resources{
	// 		"test.yaml": resource.Resources{
	// 			{
	// 				TypeName: "Individual[foo]",
	// 				State:    fooState,
	// 			},
	// 			{
	// 				TypeName: "Individual[bar]",
	// 				State:    barState,
	// 			},
	// 		},
	// 	})
	// 	resources.SetupIndividualType(t, []resources.IndividualFuncCall{
	// 		// Loading resources
	// 		{ValidateName: &resources.IndividualFuncValidateName{
	// 			Name: "foo",
	// 		}},
	// 		{ValidateName: &resources.IndividualFuncValidateName{
	// 			Name: "bar",
	// 		}},
	// 		// Loading saved host state
	// 		{ValidateName: &resources.IndividualFuncValidateName{
	// 			Name: "foo",
	// 		}},
	// 		// Reading Host State
	// 		{GetState: &resources.IndividualFuncGetState{
	// 			Name:        "foo",
	// 			ReturnState: fooState,
	// 		}},
	// 		{GetState: &resources.IndividualFuncGetState{
	// 			Name:        "bar",
	// 			ReturnState: nil,
	// 		}},
	// 		// Executing plan
	// 		{Configure: &resources.IndividualFuncConfigure{
	// 			Name:  "bar",
	// 			State: barState,
	// 		}},
	// 		{GetState: &resources.IndividualFuncGetState{
	// 			Name:        "bar",
	// 			ReturnState: barState,
	// 		}},
	// 	})
	// 	runCommand(t, Cmd{
	// 		Args:             args,
	// 		ExpectedInOutput: "Apply successful",
	// 	})
	// })

	// if t.Failed() {
	// 	return
	// }

	// t.Run("Idempotency", func(t *testing.T) {
	// 	resources.SetupIndividualType(t, []resources.IndividualFuncCall{
	// 		// Loading resources
	// 		{ValidateName: &resources.IndividualFuncValidateName{
	// 			Name: "foo",
	// 		}},
	// 		{ValidateName: &resources.IndividualFuncValidateName{
	// 			Name: "bar",
	// 		}},
	// 		// Loading saved host state
	// 		{ValidateName: &resources.IndividualFuncValidateName{
	// 			Name: "foo",
	// 		}},
	// 		{ValidateName: &resources.IndividualFuncValidateName{
	// 			Name: "bar",
	// 		}},
	// 		// Reading Host State
	// 		{GetState: &resources.IndividualFuncGetState{
	// 			Name:        "foo",
	// 			ReturnState: fooState,
	// 		}},
	// 		{GetState: &resources.IndividualFuncGetState{
	// 			Name:        "bar",
	// 			ReturnState: barState,
	// 		}},
	// 	})
	// 	runCommand(t, Cmd{
	// 		Args:             args,
	// 		ExpectedInOutput: "Apply successful",
	// 	})
	// })
}

func TestApplyIndividualAndMergeable(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	args := []string{
		"apply",
		"--log-level=trace",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
		resourcesRoot,
	}

	fooMergeableState := resources.MergeableState{
		Value: "foo",
	}

	barMergeableState := resources.MergeableState{
		Value: "bar",
	}

	fooIndividualState := resources.IndividualState{
		Value: "foo",
	}

	barIndividualState := resources.IndividualState{
		Value: "bar",
	}

	t.Run("First apply", func(t *testing.T) {
		setupBundles(t, resourcesRoot, map[string]resource.Resources{
			"a.yaml": resource.Resources{
				{
					TypeName: "Mergeable[foo]",
					State:    fooMergeableState,
				},
				{
					TypeName: "Individual[foo]",
					State:    fooIndividualState,
				},
			},
			"b.yaml": resource.Resources{
				{
					TypeName: "Mergeable[bar]",
					State:    barMergeableState,
				},
				{
					TypeName: "Individual[bar]",
					State:    barIndividualState,
				},
			},
		})
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name: "foo",
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name: "bar",
			}},
			// Executing plan
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "foo",
				State: fooIndividualState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooIndividualState,
			}},
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "bar",
				State: barIndividualState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "bar",
				ReturnState: barIndividualState,
			}},
		})
		resources.SetupMergeableType(t, []resources.MergeableFuncCall{
			// Loading resources
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetStates: &resources.MergeableFuncGetStates{
				Names: []resource.Name{
					resource.Name("foo"),
					resource.Name("bar"),
				},
				ReturnNameStateMap: map[resource.Name]resource.State{
					resource.Name("foo"): nil,
					resource.Name("bar"): nil,
				},
			}},
			// Executing plan
			{ConfigureAll: &resources.MergeableFuncConfigureAll{
				ActionNameStateMap: map[resource.Action]map[resource.Name]resource.State{
					resource.ActionConfigure: map[resource.Name]resource.State{
						resource.Name("foo"): fooMergeableState,
						resource.Name("bar"): barMergeableState,
					},
				},
			}},
			{GetStates: &resources.MergeableFuncGetStates{
				Names: []resource.Name{
					resource.Name("foo"),
					resource.Name("bar"),
				},
				ReturnNameStateMap: map[resource.Name]resource.State{
					resource.Name("foo"): fooMergeableState,
					resource.Name("bar"): barMergeableState,
				},
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
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
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
				ReturnState: fooIndividualState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "bar",
				ReturnState: barIndividualState,
			}},
		})
		resources.SetupMergeableType(t, []resources.MergeableFuncCall{
			// Loading resources
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetStates: &resources.MergeableFuncGetStates{
				Names: []resource.Name{
					resource.Name("foo"),
					resource.Name("bar"),
				},
				ReturnNameStateMap: map[resource.Name]resource.State{
					resource.Name("foo"): fooMergeableState,
					resource.Name("bar"): barMergeableState,
				},
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
			"a.yaml": resource.Resources{
				{
					TypeName: "Mergeable[foo]",
					State:    fooMergeableState,
				},
			},
			"b.yaml": resource.Resources{
				{
					TypeName: "Individual[bar]",
					State:    barIndividualState,
				},
			},
		})
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading resources
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
				Name:        "bar",
				ReturnState: barIndividualState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooIndividualState,
			}},
			// Executing plan
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "foo",
				State: nil,
			}},
		})
		resources.SetupMergeableType(t, []resources.MergeableFuncCall{
			// Loading resources
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "foo",
			}},
			// Loading saved host state
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetStates: &resources.MergeableFuncGetStates{
				Names: []resource.Name{
					resource.Name("foo"),
					resource.Name("bar"),
				},
				ReturnNameStateMap: map[resource.Name]resource.State{
					resource.Name("foo"): fooMergeableState,
					resource.Name("bar"): barMergeableState,
				},
			}},
			// Executing plan
			{ConfigureAll: &resources.MergeableFuncConfigureAll{
				ActionNameStateMap: map[resource.Action]map[resource.Name]resource.State{
					resource.ActionDestroy: map[resource.Name]resource.State{
						resource.Name("bar"): nil,
					},
				},
			}},
			{GetStates: &resources.MergeableFuncGetStates{
				Names: []resource.Name{
					resource.Name("bar"),
				},
				ReturnNameStateMap: map[resource.Name]resource.State{
					resource.Name("bar"): nil,
				},
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
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name:        "bar",
				ReturnState: barIndividualState,
			}},
		})
		resources.SetupMergeableType(t, []resources.MergeableFuncCall{
			// Loading resources
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "foo",
			}},
			// Loading saved host state
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetStates: &resources.MergeableFuncGetStates{
				Names: []resource.Name{
					resource.Name("foo"),
				},
				ReturnNameStateMap: map[resource.Name]resource.State{
					resource.Name("foo"): fooMergeableState,
				},
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
			"a.yaml": resource.Resources{
				{
					TypeName: "Mergeable[foo]",
					State:    fooMergeableState,
				},
				{
					TypeName: "Individual[foo]",
					State:    fooIndividualState,
				},
			},
			"b.yaml": resource.Resources{
				{
					TypeName: "Mergeable[bar]",
					State:    barMergeableState,
				},
				{
					TypeName: "Individual[bar]",
					State:    barIndividualState,
				},
			},
		})
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
			// Loading resources
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.IndividualFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetState: &resources.IndividualFuncGetState{
				Name: "foo",
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "bar",
				ReturnState: barIndividualState,
			}},
			// Executing plan
			{Configure: &resources.IndividualFuncConfigure{
				Name:  "foo",
				State: fooIndividualState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "foo",
				ReturnState: fooIndividualState,
			}},
		})
		resources.SetupMergeableType(t, []resources.MergeableFuncCall{
			// Loading resources
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "foo",
			}},
			// Reading Host State
			{GetStates: &resources.MergeableFuncGetStates{
				Names: []resource.Name{
					resource.Name("foo"),
					resource.Name("bar"),
				},
				ReturnNameStateMap: map[resource.Name]resource.State{
					resource.Name("foo"): fooMergeableState,
					resource.Name("bar"): nil,
				},
			}},
			// Executing plan
			{ConfigureAll: &resources.MergeableFuncConfigureAll{
				ActionNameStateMap: map[resource.Action]map[resource.Name]resource.State{
					resource.ActionConfigure: map[resource.Name]resource.State{
						resource.Name("bar"): barMergeableState,
					},
				},
			}},
			{GetStates: &resources.MergeableFuncGetStates{
				Names: []resource.Name{
					resource.Name("bar"),
				},
				ReturnNameStateMap: map[resource.Name]resource.State{
					resource.Name("bar"): barMergeableState,
				},
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
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
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
				ReturnState: fooIndividualState,
			}},
			{GetState: &resources.IndividualFuncGetState{
				Name:        "bar",
				ReturnState: barIndividualState,
			}},
		})
		resources.SetupMergeableType(t, []resources.MergeableFuncCall{
			// Loading resources
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "bar",
			}},
			// Loading saved host state
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "foo",
			}},
			{ValidateName: &resources.MergeableFuncValidateName{
				Name: "bar",
			}},
			// Reading Host State
			{GetStates: &resources.MergeableFuncGetStates{
				Names: []resource.Name{
					resource.Name("foo"),
					resource.Name("bar"),
				},
				ReturnNameStateMap: map[resource.Name]resource.State{
					resource.Name("foo"): fooMergeableState,
					resource.Name("bar"): barMergeableState,
				},
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
		"--log-level=trace",
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
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
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
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
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
			ExpectedInOutput: "Previous host state is not clean:",
		})
	})
}

func TestApplyFailureWithRollback(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	args := []string{
		"apply",
		"--log-level=trace",
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
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
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
		resources.SetupIndividualTypeMock(t, []resources.IndividualFuncCall{
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
