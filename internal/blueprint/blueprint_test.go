package blueprint

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/resources"
)

func TestStepYaml(t *testing.T) {
	t.Run("SingleResource", func(t *testing.T) {
		testStep := NewSingleResourceStep(&resources.File{
			Path: "/tmp/foo",
			Perm: 0644,
		})

		testStepYamlStr := `single_resource_type: File
single_resource:
    path: /tmp/foo
    perm: 420
`

		t.Run("Marshal", func(t *testing.T) {
			bs, err := yaml.Marshal(testStep)
			require.NoError(t, err)

			require.Equal(t, testStepYamlStr, string(bs))
		})

		t.Run("Unmarshal", func(t *testing.T) {
			unmarshaledStep := Step{}
			err := yaml.Unmarshal([]byte(testStepYamlStr), &unmarshaledStep)
			require.NoError(t, err)
			require.Equal(t, testStep, &unmarshaledStep)
		})
	})

	t.Run("GroupResource", func(t *testing.T) {
		testStep := NewGroupResourceStep(&resources.APTPackages{})
		testStep.AppendGroupResource(&resources.APTPackage{
			Package: "foo",
		})
		testStep.AppendGroupResource(&resources.APTPackage{
			Package: "bar",
		})

		testStepYamlStr := `group_resource_type: APTPackages
group_resources_type: APTPackage
group_resources:
    - APTPackage:
        package: foo
    - APTPackage:
        package: bar
`

		t.Run("Marshal", func(t *testing.T) {
			bs, err := yaml.Marshal(testStep)
			require.NoError(t, err)

			require.Equal(t, testStepYamlStr, string(bs))
		})

		t.Run("Unmarshal", func(t *testing.T) {
			unmarshaledStep := Step{}
			err := yaml.Unmarshal([]byte(testStepYamlStr), &unmarshaledStep)
			require.NoError(t, err)
			require.Equal(t, testStep, &unmarshaledStep)
		})
	})
}
