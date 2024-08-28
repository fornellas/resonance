package resources

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestValidateResource(t *testing.T) {
	type testCase struct {
		name          string
		resource      Resource
		errorContains string
	}
	for _, tc := range []testCase{
		{
			name: "valid",
			resource: &File{
				Path: "/tmp/foo",
			},
		},
		{
			name:          "missing id",
			resource:      &File{},
			errorContains: "resource id field \"path\" must be set",
		},
		{
			name: "absent with other fields set",
			resource: &File{
				Path:   "/tmp/foo",
				Absent: true,
				Perm:   0644,
			},
			errorContains: "resource has absent set to true, but other fields are set",
		},
		{
			name: "invalid state",
			resource: &File{
				Path: "foo",
			},
			errorContains: "'path' must be absolute",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateResource(tc.resource)
			if tc.errorContains == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.errorContains)
			}
		})
	}
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

func TestGetResourceTypeName(t *testing.T) {
	require.Equal(t, "File", GetResourceTypeName(&File{Path: "/foo"}))
}

func TestGetResourceTypeId(t *testing.T) {
	require.Equal(t, "File:/foo", GetResourceTypeId(&File{Path: "/foo"}))
}

func TestGetResourceYaml(t *testing.T) {
	require.Equal(t, "path: /foo", GetResourceYaml(&File{Path: "/foo"}))
}

func TestHashResource(t *testing.T) {
	hash := HashResource(&File{Path: "/foo"})

	require.Len(t, hash, 64)

	_, err := hex.DecodeString(hash)
	require.NoError(t, err)

	require.Equal(
		t,
		HashResource(&File{Path: "/foo"}),
		HashResource(&File{Path: "/foo", Uid: 33, Gid: 33}),
	)

	require.NotEqual(
		t,
		HashResource(&File{Path: "/foo"}),
		HashResource(&File{Path: "/bar"}),
	)
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

func TestNewResourceWithSameId(t *testing.T) {
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
			NewResourceWithSameId(tc.Resource),
		)
	}
}

func TestSetResourceAbsent(t *testing.T) {
	resource := &File{
		Path: "/tmp/foo",
	}
	SetResourceAbsent(resource)
	require.True(t, resource.Absent)
}

func TestGetResourceAbsent(t *testing.T) {
	resource := &File{
		Path: "/tmp/foo",
	}
	require.False(t, GetResourceAbsent(resource))
	resource.Absent = true
	require.True(t, GetResourceAbsent(resource))
}

func TestSatisfies(t *testing.T) {
	require.True(t, Satisfies(
		&File{
			Path: "foo",
		},
		&File{
			Path: "foo",
		},
	))

	require.False(t, Satisfies(
		&File{
			Path: "foo",
		},
		&File{
			Path: "foo",
			Perm: 0644,
		},
	))

	require.True(t, Satisfies(
		&APTPackage{
			Version: "1",
		},
		&APTPackage{
			Version: "1",
		},
	))

	require.True(t, Satisfies(
		&APTPackage{
			Version: "1",
		},
		&APTPackage{},
	))

	require.False(t, Satisfies(
		&APTPackage{},
		&APTPackage{
			Version: "1",
		},
	))

	require.False(t, Satisfies(
		&APTPackage{
			Version: "1",
		},
		&APTPackage{
			Absent: true,
		},
	))
}

func TestResourceMap(t *testing.T) {
	file := &File{
		Path: "/foo",
		Perm: 0644,
	}
	aptPackage := &APTPackage{
		Package: "bash",
		Version: "1",
	}
	resourceMap := NewResourceMap(Resources{file, aptPackage})

	require.True(t, reflect.DeepEqual(
		file,
		resourceMap.GetResourceWithSameTypeId(&File{
			Path: "/foo",
			Perm: 0777,
		}),
	))

	require.True(t, reflect.DeepEqual(
		aptPackage,
		resourceMap.GetResourceWithSameTypeId(&APTPackage{
			Package: "bash",
			Version: "2",
		}),
	))
}

func TestGetResourcesYaml(t *testing.T) {
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
    perm: 420
- APTPackage:
    package: foo`

	require.Equal(t, testResourcesYamlStr, GetResourcesYaml(testResources))
}

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
    perm: 420
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

func TestNewResourcesWithSameIds(t *testing.T) {
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
		NewResourcesWithSameIds(
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

func TestGetGroupResourceTypeName(t *testing.T) {
	require.Equal(
		t,
		"APTPackages",
		GetGroupResourceTypeName(&APTPackages{}),
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
