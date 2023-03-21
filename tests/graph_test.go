package tests

import (
	"testing"

	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/tests/resources"
)

func TestGraph(t *testing.T) {
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

	t.Run("plan link", func(t *testing.T) {
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
			"graph",
			"--log-level=debug",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
			"--format=link",
			resourcesRoot,
		}
		runCommand(t, Cmd{
			Args:             args,
			ExpectedInStdout: "http://magjac.com/graphviz-visual-editor/?dot=digraph+resonance+%7B%0A++node+%5Bshape%3Dbox%5D+%22Test%5B%F0%9F%94%A7+foo%5D%22%0A++node+%5Bshape%3Dbox%5D+%22Test%5B%E2%9C%85+bar%5D%22%0A++%22Test%5B%F0%9F%94%A7+foo%5D%22+-%3E+%22Test%5B%E2%9C%85+bar%5D%22%0A%7D%0A",
		})
	})

	t.Run("plan dot", func(t *testing.T) {
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
			"graph",
			"--log-level=debug",
			"--force-color",
			"--localhost",
			"--state-root", stateRoot,
			"--format=dot",
			resourcesRoot,
		}
		runCommand(t, Cmd{
			Args: args,
			ExpectedInStdout: (`digraph resonance {` + "\n" +
				`  node [shape=box] "Test[🔧 foo]"` + "\n" +
				`  node [shape=box] "Test[✅ bar]"` + "\n" +
				`  "Test[🔧 foo]" -> "Test[✅ bar]"` + "\n" +
				`}` + "\n"),
		})
	})

}
