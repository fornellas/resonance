package tests

import (
	"testing"

	"github.com/fornellas/resonance/resource"
)

func TestValidate(t *testing.T) {
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
	})
	args := []string{
		"validate",
		"--log-level=debug",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
		resourcesRoot,
	}
	runCommand(t, Cmd{
		Args:             args,
		ExpectedInOutput: "Validation successful",
	})
}
