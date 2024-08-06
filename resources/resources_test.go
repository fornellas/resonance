package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestResourcesYAML(t *testing.T) {
	testResources := Resources{
		&File{
			Path: "/tmp/foo",
			Perm: 0644,
		},
		&APTPackage{
			Package: "foo",
		},
	}

	testResourcesYamlStr := `- File:
    path: /tmp/foo
    content: ""
    perm: 420
    uid: 0
    gid: 0
- APTPackage:
    package: foo
`

	t.Run("Marshal", func(t *testing.T) {
		bs, err := yaml.Marshal(testResources)
		require.NoError(t, err)

		require.Equal(t, testResourcesYamlStr, string(bs))
	})

	t.Run("Unmarshal", func(t *testing.T) {
		unmarshaledResources := Resources{}
		err := yaml.Unmarshal([]byte(testResourcesYamlStr), &unmarshaledResources)
		require.NoError(t, err)
		require.Equal(t, testResources, unmarshaledResources)
	})
}

func TestGetResourceId(t *testing.T) {
	type TestCase struct {
		Resource Resource
		Id       string
	}

	for _, tc := range []TestCase{
		{
			Resource: &File{
				Path: "/tmp/foo",
				Perm: 0644,
			},
			Id: "/tmp/foo",
		},
		{
			Resource: &APTPackage{
				Package: "foo",
				Version: "1",
			},
			Id: "foo",
		},
	} {
		require.Equal(t, tc.Id, GetResourceId(tc.Resource))
	}
}

func TestGetResourceByTypeName(t *testing.T) {
	type TestCase struct {
		Name     string
		Resource Resource
	}

	for _, tc := range []TestCase{
		{
			Name:     "File",
			Resource: &File{},
		},
		{
			Name:     "APTPackage",
			Resource: &APTPackage{},
		},
	} {
		resource := GetResourceByTypeName(tc.Name)
		require.Equal(t, tc.Resource, resource)
	}
}

func TestGetResourceTypeNames(t *testing.T) {
	require.Equal(t, []string{"APTPackage", "File"}, GetResourceTypeNames())
}

func TestNewResourceCopyWithOnlyId(t *testing.T) {
	type TestCase struct {
		Resource               Resource
		ResourceCopyWithOnlyId Resource
	}

	for _, tc := range []TestCase{
		{
			Resource: &File{
				Path: "/tmp/foo",
				Perm: 0644,
			},
			ResourceCopyWithOnlyId: &File{
				Path: "/tmp/foo",
			},
		},
		{
			Resource: &APTPackage{
				Package: "foo",
				Version: "1",
			},
			ResourceCopyWithOnlyId: &APTPackage{
				Package: "foo",
			},
		},
	} {
		require.Equal(
			t,
			tc.ResourceCopyWithOnlyId,
			NewResourceCopyWithOnlyId(tc.Resource),
		)
	}
}

func TestNewResourcesCopyWithOnlyId(t *testing.T) {
	require.Equal(
		t,
		Resources{
			&File{
				Path: "/tmp/foo",
			},
			&APTPackage{
				Package: "foo",
			},
		},
		NewResourcesCopyWithOnlyId(
			Resources{
				&File{
					Path: "/tmp/foo",
					Perm: 0644,
				},
				&APTPackage{
					Package: "foo",
					Version: "1",
				},
			},
		),
	)
}

func TestGetSingleResourceByTypeName(t *testing.T) {
	require.Equal(
		t,
		&File{},
		GetSingleResourceByTypeName("File"),
	)
}

func TestGetGroupResourceByTypeName(t *testing.T) {
	require.Equal(
		t,
		&APTPackages{},
		GetGroupResourceByTypeName("APTPackage"),
	)
}

func TestIsGroupResource(t *testing.T) {
	require.False(t, IsGroupResource("File"))
	require.True(t, IsGroupResource("APTPackage"))
}
