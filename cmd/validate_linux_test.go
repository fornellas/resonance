package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/cmd/test/resources"
	"github.com/fornellas/resonance/internal/resource"
)

func TestValidate(t *testing.T) {
	resourcesRoot := filepath.Join(t.TempDir(), "state")
	require.NoError(t, os.Mkdir(resourcesRoot, os.FileMode(0700)))

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
		"--localhost",
		resourcesRoot,
	}
	cmd := TestCmd{
		Args:             args,
		ExpectedInOutput: "Validation successful",
	}
	cmd.Run(t)
}
