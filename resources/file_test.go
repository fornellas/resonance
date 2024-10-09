package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	type testCase struct {
		Title         string
		File          File
		ErrorContains string
	}
	user := "user"
	group := "group"
	t.Run("Validate()", func(t *testing.T) {
		for _, tc := range []testCase{
			{
				Title: "valid path",
				File: File{
					Path:   "/foo",
					Absent: true,
				},
			},
			{
				Title: "relative path",
				File: File{
					Path:   "foo",
					Absent: true,
				},
				ErrorContains: "'path' must be absolute",
			},
			{
				Title: "absent",
				File: File{
					Path:   "/foo",
					Absent: true,
				},
			},
			{
				Title: "absent with type definition",
				File: File{
					Path:         "/foo",
					Absent:       true,
					SymbolicLink: "/bar",
				},
				ErrorContains: "can not set absent and a file type at the same time",
			},
			{
				Title: "socket",
				File: File{
					Path:   "/foo",
					Socket: true,
				},
			},
			{
				Title: "symlink",
				File: File{
					Path:         "/foo",
					SymbolicLink: "/bar",
				},
			},
			{
				Title: "regular file",
				File: File{
					Path:        "/foo",
					RegularFile: new(string),
				},
			},
			{
				Title: "block device",
				File: File{
					Path:        "/foo",
					BlockDevice: new(uint64),
				},
			},
			{
				Title: "directory",
				File: File{
					Path:      "/foo",
					Directory: &[]File{},
				},
			},
			{
				Title: "char device",
				File: File{
					Path:            "/foo",
					CharacterDevice: new(uint64),
				},
			},
			{
				Title: "fifo",
				File: File{
					Path: "/foo",
					FIFO: true,
				},
			},
			{
				Title: "multiple types",
				File: File{
					Path:   "/foo",
					FIFO:   true,
					Socket: true,
				},
				ErrorContains: "only one file type can be defined",
			},
			{
				Title: "uid",
				File: File{
					Path:        "/foo",
					RegularFile: new(string),
					Uid:         new(uint32),
				},
			},
			{
				Title: "user",
				File: File{
					Path:        "/foo",
					RegularFile: new(string),
					User:        &user,
				},
			},
			{
				Title: "uid + user",
				File: File{
					Path:        "/foo",
					RegularFile: new(string),
					Uid:         new(uint32),
					User:        &user,
				},
				ErrorContains: "can't set both 'uid' and 'user'",
			},
			{
				Title: "gid",
				File: File{
					Path:        "/foo",
					RegularFile: new(string),
					Gid:         new(uint32),
				},
			},
			{
				Title: "group",
				File: File{
					Path:        "/foo",
					RegularFile: new(string),
					Group:       &group,
				},
			},
			{
				Title: "gid	+ group",
				File: File{
					Path:        "/foo",
					RegularFile: new(string),
					Gid:         new(uint32),
					Group:       &group,
				},
				ErrorContains: "can't set both 'gid' and 'group'",
			},
		} {
			t.Run(tc.Title, func(t *testing.T) {
				err := tc.File.Validate()
				if tc.ErrorContains != "" {
					require.ErrorContains(t, err, tc.ErrorContains)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
	t.Run("Load()", func(t *testing.T) {

	})
	t.Run("Resolve()", func(t *testing.T) {

	})
	t.Run("Apply()", func(t *testing.T) {
		// TODO after apply, load and reflect.DeepEqual to validate
	})
}
