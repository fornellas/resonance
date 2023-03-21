package tests

import (
	"testing"

	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/tests/resources"
)

func TestPlan(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	fooState := resources.TestState{
		Value: "foo",
	}

	barState := resources.TestState{
		Value: "bar",
	}

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
			ReturnState: barState,
		}},
	})
	args := []string{
		"plan",
		"--log-level=debug",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
		resourcesRoot,
	}
	runCommand(t, Cmd{
		Args: args,
		ExpectedInOutput: ("  🔧 Test[foo]\n" +
			"    value: foo\n" +
			"    \n" +
			"  Test[✅ bar]\n"),
	})
}
