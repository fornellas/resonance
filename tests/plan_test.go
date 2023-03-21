package tests

import (
	"testing"

	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/tests/resources"
)

func TestPlan(t *testing.T) {
	stateRoot, resourcesRoot := setupDirs(t)

	fooState := resources.IndividualState{
		Value: "foo",
	}

	barState := resources.IndividualState{
		Value: "bar",
	}

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
		ExpectedInOutput: ("  🔧 Individual[foo]\n" +
			"    value: foo\n" +
			"    \n" +
			"  Individual[✅ bar]\n"),
	})
}
