package main

import (
	"path/filepath"
	"testing"

	"github.com/fornellas/resonance/cmd/test/resources"
	"github.com/fornellas/resonance/internal/resource"
)

func TestValidate(t *testing.T) {
	resourcesRoot := filepath.Join(t.TempDir(), "resources")

	CreateResourceYamls(t, resourcesRoot, map[string]resource.ResourceDefs{
		"test.yaml": resource.ResourceDefs{
			{
				TypeName: "Individual[foo]",
				State: resources.IndividualState{
					Value: "foo",
				},
			},
			{
				TypeName: "Individual[bar]",
				State: resources.IndividualState{
					Value: "bar",
				},
			},
		},
	})

	resources.SetupIndividualTypeMock(t, []resources.IndividualFnCall{
		{ValidateName: &resources.IndividualFnValidateName{
			Name: "foo",
		}},
		{ValidateName: &resources.IndividualFnValidateName{
			Name: "bar",
		}},
	})

	cmd := TestCmd{
		Args: []string{
			"validate",
			"--localhost",
			resourcesRoot,
		},
		ExpectStderrContains: "Validation successful",
	}
	cmd.Run(t)
}
