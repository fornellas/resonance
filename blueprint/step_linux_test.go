package blueprint

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/fornellas/slogxt/log"

	hostPkg "github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

func TestStepMisc(t *testing.T) {
	type testCase struct {
		name           string
		resourceType   string
		singleResource resourcesPkg.SingleResource
		groupResource  resourcesPkg.GroupResource
		groupResources resourcesPkg.Resources
		string         string
		detailedString string
		yaml           string
	}

	var mode types.FileMode = 0644
	testCases := []testCase{
		{
			name:         "SingleResource",
			resourceType: "File",
			singleResource: &resourcesPkg.File{
				Path: "/tmp/foo",
				Mode: &mode,
			},
			string: "File:/tmp/foo",
			detailedString: `File:
  path: /tmp/foo
  mode: "0644"`,
			yaml: `single_resource_type: File
single_resource:
    path: /tmp/foo
    mode: "0644"
`,
		},
		{
			name:          "GroupResource",
			resourceType:  "APTPackages",
			groupResource: &resourcesPkg.APTPackages{},
			groupResources: resourcesPkg.Resources{
				&resourcesPkg.APTPackage{
					Package: "bar",
					Version: "2",
				},
				&resourcesPkg.APTPackage{
					Package: "foo",
					Version: "1",
				},
			},
			string: "APTPackages:bar,foo",
			detailedString: `APTPackages:
  - package: bar
    version: "2"
  - package: foo
    version: "1"`,
			yaml: `group_resource_type: APTPackages
group_resources_type: APTPackage
group_resources:
    - APTPackage:
        package: bar
        version: "2"
    - APTPackage:
        package: foo
        version: "1"
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var step *Step
			if tc.singleResource != nil {
				step = NewSingleResourceStep(tc.singleResource)
				require.True(t, step.IsSingleResource())
				require.False(t, step.IsGroupResource())
				require.Equal(t, resourcesPkg.Resources{tc.singleResource}, step.Resources())
			} else if tc.groupResource != nil {
				step = NewGroupResourceStep(tc.groupResource)
				for _, resource := range tc.groupResources {
					step.AppendGroupResource(resource)
				}
				require.False(t, step.IsSingleResource())
				require.Equal(t, tc.groupResource, step.MustGroupResource())
				require.Equal(t, tc.groupResources, step.Resources())
			} else {
				panic("bug: bad test case")
			}
			require.NotNil(t, step)

			require.Equal(t, tc.resourceType, step.Type())
			require.Equal(t, tc.string, step.String())
			require.Equal(t, tc.detailedString, step.DetailedString())

			bs, err := yaml.Marshal(step)
			require.NoError(t, err)
			require.Equal(t, tc.yaml, string(bs))

			unmarshaledStep := Step{}
			err = yaml.Unmarshal([]byte(tc.yaml), &unmarshaledStep)
			require.NoError(t, err)
			require.Equal(t, step, &unmarshaledStep)
		})
	}
}

func TestStepResolve(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)
	localhost := hostPkg.Local{}

	fileContent := "foo"
	user := "root"
	step := NewSingleResourceStep(&resourcesPkg.File{
		Path:        "/bin",
		RegularFile: &fileContent,
		User:        &user,
	})
	step.Resolve(ctx, localhost)
	require.Equal(t,
		&resourcesPkg.File{
			Path:        "/bin",
			RegularFile: &fileContent,
			Uid:         new(uint32),
			Gid:         new(uint32),
		},
		step.Resources()[0],
	)

	step = NewGroupResourceStep(&resourcesPkg.APTPackages{})
	step.AppendGroupResource(&resourcesPkg.APTPackage{
		Package: "foo",
	})
	step.AppendGroupResource(&resourcesPkg.APTPackage{
		Package: "bar",
	})
	step.Resolve(ctx, localhost)
	require.Equal(t,
		resourcesPkg.Resources{
			&resourcesPkg.APTPackage{
				Package: "bar",
			},
			&resourcesPkg.APTPackage{
				Package: "foo",
			},
		},
		step.Resources(),
	)
}
