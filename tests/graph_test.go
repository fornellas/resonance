package tests

import (
	"testing"

	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/tests/resources"
)

func TestGraph(t *testing.T) {
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

	t.Run("plan link", func(t *testing.T) {
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
			"graph",
			"--log-level=trace",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
			"--format=link",
			resourcesRoot,
		}
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInStdout: "http://magjac.com/graphviz-visual-editor/?dot=digraph+resonance+%7B%0A++node+%5Bshape%3Dbox%5D+%22%F0%9F%94%A7+Individual%5Bfoo%5D%22%0A++node+%5Bshape%3Dbox%5D+%22%E2%9C%85+Individual%5Bbar%5D%22%0A++%22%F0%9F%94%A7+Individual%5Bfoo%5D%22+-%3E+%22%E2%9C%85+Individual%5Bbar%5D%22%0A%7D%0A",
		})
	})

	t.Run("plan dot", func(t *testing.T) {
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
			"graph",
			"--log-level=trace",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
			"--format=dot",
			resourcesRoot,
		}
		runCommand(t, Cmd{
			Args: args,
			ExpectedInStdout: (`digraph resonance {` + "\n" +
				`  node [shape=box] "🔧 Individual[foo]"` + "\n" +
				`  node [shape=box] "✅ Individual[bar]"` + "\n" +
				`  "🔧 Individual[foo]" -> "✅ Individual[bar]"` + "\n" +
				`}` + "\n"),
		})
	})

}
