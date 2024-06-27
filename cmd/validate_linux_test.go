package main

import (
	"testing"

	"github.com/fornellas/resonance/cmd/test/resources"
	"github.com/fornellas/resonance/resource"
)

func TestValidate(t *testing.T) {
	stateRoot, resourcesRoot := SetupDirs(t)

	fooState := resources.IndividualState{
		Value: "foo",
	}

	barState := resources.IndividualState{
		Value: "bar",
	}

	SetupBundles(t, resourcesRoot, map[string]resource.Resources{
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
	})
	args := []string{
		"validate",
		"--log-level=trace",
		"--force-color",
		"--localhost",
		"--state-root", stateRoot,
		resourcesRoot,
	}
	cmd := TestCmd{
		Args:             args,
		ExpectedInOutput: "Validation successful",
	}
	cmd.Run(t)
}
