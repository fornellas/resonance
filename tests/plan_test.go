package tests

import (
	"testing"

	"github.com/fornellas/resonance/resource"
)

func TestPlan(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	fooState := TestState{
		Value: "foo",
	}

	barState := TestState{
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
		ExpectedInOutput: ("📝 Plan\n" +
			"  🔧 Test[foo]\n" +
			"    value: foo\n" +
			"    \n" +
			"  Test[✅ bar]\n" +
			"🎆 Success"),
	})
}
