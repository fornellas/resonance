package resources

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"github.com/fornellas/slogxt/log"
	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
)

func isRoot(t *testing.T) bool {
	u, err := user.Current()
	require.NoError(t, err)
	return u.Uid == "0"
}

func TestLoadFile(t *testing.T) {
	ctx := log.WithTestLogger(t.Context())
	hst := host.Local{}

	var uid uint32 = uint32(os.Getuid())
	var gid uint32 = uint32(os.Getgid())
	var mode types.FileMode = 01724
	contents := "foo\nbar"
	var device types.FileDevice = 234122345

	t.Run("existing", func(t *testing.T) {
		prefix := t.TempDir()
		require.NoError(t, syscall.Chmod(prefix, uint32(mode)))

		socketPath := filepath.Join(prefix, "socket")
		require.NoError(t, syscall.Mknod(socketPath, syscall.S_IFSOCK, 0))
		require.NoError(t, syscall.Chmod(socketPath, uint32(mode)))

		symlinkPath := filepath.Join(prefix, "symlink")
		symlinkOldPath := "target"
		require.NoError(t, syscall.Symlink(symlinkOldPath, symlinkPath))

		regularPath := filepath.Join(prefix, "regular")
		require.NoError(t, os.WriteFile(regularPath, []byte(contents), os.FileMode(mode)))
		require.NoError(t, syscall.Chmod(regularPath, uint32(mode)))

		blockPath := filepath.Join(prefix, "block")
		characterPath := filepath.Join(prefix, "character")
		if isRoot(t) {
			require.NoError(t, syscall.Mknod(blockPath, syscall.S_IFBLK|uint32(mode), int(device)))
			require.NoError(t, syscall.Chmod(blockPath, uint32(mode)))

			require.NoError(t, syscall.Mknod(characterPath, syscall.S_IFCHR|uint32(mode), int(device)))
			require.NoError(t, syscall.Chmod(characterPath, uint32(mode)))
		}

		fifoPath := filepath.Join(prefix, "fifo")
		require.NoError(t, syscall.Mknod(fifoPath, syscall.S_IFIFO|uint32(mode), 0))
		require.NoError(t, syscall.Chmod(fifoPath, uint32(mode)))

		file, err := LoadFile(ctx, hst, prefix)
		require.NoError(t, err)

		expectedFile := File{
			Path: prefix,
			Directory: &[]*File{
				{
					Path: fifoPath,
					FIFO: true,
					Mode: &mode,
					Uid:  &uid,
					Gid:  &gid,
				},
				{
					Path:        regularPath,
					RegularFile: &contents,
					Mode:        &mode,
					Uid:         &uid,
					Gid:         &gid,
				},
				{
					Path:   socketPath,
					Socket: true,
					Mode:   &mode,
					Uid:    &uid,
					Gid:    &gid,
				},
				{
					Path:         symlinkPath,
					SymbolicLink: symlinkOldPath,
					Uid:          &uid,
					Gid:          &gid,
				},
			},
			Mode: &mode,
			Uid:  &uid,
			Gid:  &gid,
		}

		if isRoot(t) {
			*expectedFile.Directory = append(
				[]*File{
					{
						Path:        blockPath,
						BlockDevice: &device,
						Mode:        &mode,
						Uid:         &uid,
						Gid:         &gid,
					},
					{
						Path:            characterPath,
						CharacterDevice: &device,
						Mode:            &mode,
						Uid:             &uid,
						Gid:             &gid,
					},
				},
				*expectedFile.Directory...,
			)
		}

		require.True(t, reflect.DeepEqual(expectedFile, file))
	})
	t.Run("absent", func(t *testing.T) {
		prefix := t.TempDir()
		path := filepath.Join(prefix, "absent")
		file, err := LoadFile(ctx, hst, path)
		require.NoError(t, err)
		require.NoError(t, err)
		require.True(t, reflect.DeepEqual(file, File{
			Path:   path,
			Absent: true,
		}))
	})
}

func TestFile_Satisfies(t *testing.T) {
	ctx := log.WithTestLogger(t.Context())
	hst := host.Local{}

	user := "root"
	var uid uint32 = uint32(os.Getuid())
	group := "root"
	var gid uint32 = uint32(os.Getgid())
	var mode1 types.FileMode = 0644
	var mode2 types.FileMode = 0755
	var device1 types.FileDevice = 12345
	var device2 types.FileDevice = 67890
	contents1 := "content1"
	contents2 := "content2"

	type TestCase struct {
		Title             string
		File              File
		OtherFile         File
		ExpectedSatisfies bool
		ErrorContains     string
	}

	for _, tc := range []TestCase{
		// Path
		{
			Title: "different paths",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/bar",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: false,
		},
		{
			Title: "same path",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: true,
		},
		// Absent
		{
			Title: "other wants absent but file is not",
			File: File{
				Path:   "/foo",
				Absent: false,
			},
			OtherFile: File{
				Path:   "/foo",
				Absent: true,
			},
			ExpectedSatisfies: false,
		},
		{
			Title: "both absent",
			File: File{
				Path:   "/foo",
				Absent: true,
			},
			OtherFile: File{
				Path:   "/foo",
				Absent: true,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "file is absent but other is not",
			File: File{
				Path:   "/foo",
				Absent: true,
			},
			OtherFile: File{
				Path:   "/foo",
				Absent: false,
			},
			ExpectedSatisfies: true,
		},
		// Socket
		{
			Title: "same socket",
			File: File{
				Path:   "/foo",
				Socket: true,
			},
			OtherFile: File{
				Path:   "/foo",
				Socket: true,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "different types - socket vs regular",
			File: File{
				Path:   "/foo",
				Socket: true,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: false,
		},
		// SymbolicLink
		{
			Title: "same symbolic links",
			File: File{
				Path:         "/foo",
				SymbolicLink: "/target1",
			},
			OtherFile: File{
				Path:         "/foo",
				SymbolicLink: "/target1",
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "symbolic links to different targets",
			File: File{
				Path:         "/foo",
				SymbolicLink: "/target1",
			},
			OtherFile: File{
				Path:         "/foo",
				SymbolicLink: "/target2",
			},
			ExpectedSatisfies: false,
		},
		// RegularFile
		{
			Title: "regular files with same contents",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "regular files with different contents",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents2,
			},
			ExpectedSatisfies: false,
		},
		// BlockDevice
		{
			Title: "same block devices",
			File: File{
				Path:        "/foo",
				BlockDevice: &device1,
			},
			OtherFile: File{
				Path:        "/foo",
				BlockDevice: &device1,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "block devices with different major/minor",
			File: File{
				Path:        "/foo",
				BlockDevice: &device1,
			},
			OtherFile: File{
				Path:        "/foo",
				BlockDevice: &device2,
			},
			ExpectedSatisfies: false,
		},
		// Directory
		{
			Title: "directory with contents",
			File: File{
				Path: "/foo",
				Directory: &[]*File{
					{
						Path:        "/foo/bar",
						RegularFile: &contents1,
					},
				},
			},
			OtherFile: File{
				Path: "/foo",
				Directory: &[]*File{
					{
						Path:        "/foo/bar",
						RegularFile: &contents1,
					},
				},
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "directory with different contents",
			File: File{
				Path: "/foo",
				Directory: &[]*File{
					{
						Path:        "/foo/bar",
						RegularFile: &contents1,
					},
				},
			},
			OtherFile: File{
				Path: "/foo",
				Directory: &[]*File{
					{
						Path:        "/foo/bar",
						RegularFile: &contents2,
					},
				},
			},
			ExpectedSatisfies: false,
		},
		// CharacterDevice
		{
			Title: "same character devices",
			File: File{
				Path:            "/foo",
				CharacterDevice: &device1,
			},
			OtherFile: File{
				Path:            "/foo",
				CharacterDevice: &device1,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "character devices with different major/minor",
			File: File{
				Path:            "/foo",
				CharacterDevice: &device1,
			},
			OtherFile: File{
				Path:            "/foo",
				CharacterDevice: &device2,
			},
			ExpectedSatisfies: false,
		},
		// FIFO
		{
			Title: "same FIFO",
			File: File{
				Path: "/foo",
				FIFO: true,
			},
			OtherFile: File{
				Path: "/foo",
				FIFO: true,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "different types - FIFO vs socket",
			File: File{
				Path: "/foo",
				FIFO: true,
			},
			OtherFile: File{
				Path:   "/foo",
				Socket: true,
			},
			ExpectedSatisfies: false,
		},
		// Mode
		{
			Title: "other has mode but file doesn't",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Mode:        &mode1,
			},
			ExpectedSatisfies: false,
		},
		{
			Title: "different modes",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Mode:        &mode1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Mode:        &mode2,
			},
			ExpectedSatisfies: false,
		},
		{
			Title: "same modes",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Mode:        &mode1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Mode:        &mode1,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "file has mode but other doesn't",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Mode:        &mode1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: true,
		},
		// User
		{
			Title: "other has user but file doesn't",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				User:        &user,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "file has user but other doesn't",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				User:        &user,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "file and other doesn't have user",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "file has user and other has same user",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				User:        &user,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				User:        &user,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "file and other have no user",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: true,
		},
		// Uid
		{
			Title: "other has uid but file doesn't",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Uid:         &uid,
			},
			ExpectedSatisfies: false,
		},
		{
			Title: "different uids",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Uid:         &uid,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Uid:         func() *uint32 { v := uid + 1; return &v }(),
			},
			ExpectedSatisfies: false,
		},
		{
			Title: "same uids",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Uid:         &uid,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Uid:         &uid,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "file has uid but other doesn't",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Uid:         &uid,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: false,
		},
		// Group
		{
			Title: "other has group but file doesn't",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Group:       &group,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "file has group but other doesn't",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Group:       &group,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "file and other doesn't have group",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "file has group and other has same group",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Group:       &group,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Group:       &group,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "file and other have no group",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: true,
		},
		// Gid
		{
			Title: "other has gid but file doesn't",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Gid:         &gid,
			},
			ExpectedSatisfies: false,
		},
		{
			Title: "different gids",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Gid:         &gid,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Gid:         func() *uint32 { v := gid + 1; return &v }(),
			},
			ExpectedSatisfies: false,
		},
		{
			Title: "same gids",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Gid:         &gid,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Gid:         &gid,
			},
			ExpectedSatisfies: true,
		},
		{
			Title: "file has gid but other doesn't",
			File: File{
				Path:        "/foo",
				RegularFile: &contents1,
				Gid:         &gid,
			},
			OtherFile: File{
				Path:        "/foo",
				RegularFile: &contents1,
			},
			ExpectedSatisfies: false,
		},
	} {
		t.Run(tc.Title, func(t *testing.T) {
			satisfies, err := tc.File.Satisfies(ctx, hst, &tc.OtherFile)
			if tc.ErrorContains != "" {
				require.ErrorContains(t, err, tc.ErrorContains)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.ExpectedSatisfies, satisfies)
			}
		})
	}
}

func TestFile_Validate(t *testing.T) {
	user := "user"
	var uid uint32 = uint32(os.Getuid())
	group := "group"
	var gid uint32 = uint32(os.Getgid())
	var mode types.FileMode = 01724
	var badMode types.FileMode = types.FileModeBitsMask + 1
	var device types.FileDevice = 234122345
	type TestCase struct {
		Title         string
		File          File
		ErrorContains string
	}
	for _, tc := range []TestCase{
		// Path
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
			ErrorContains: "'path' must be an absolute unix path",
		},
		{
			Title: "not clean path",
			File: File{
				Path:   "//foo",
				Absent: true,
			},
			ErrorContains: "'path' must be a clean unix path",
		},
		{
			Title: "windows path",
			File: File{
				Path:   "C:\\foo",
				Absent: true,
			},
			ErrorContains: "'path' must be an absolute unix path",
		},
		// Absent / Type
		{
			Title: "absent",
			File: File{
				Path:   "/foo",
				Absent: true,
			},
		},
		{
			Title: "absent with mode",
			File: File{
				Path:   "/foo",
				Absent: true,
				Mode:   &mode,
			},
			ErrorContains: "'mode' can not be set with 'absent'",
		},
		{
			Title: "absent with uid",
			File: File{
				Path:   "/foo",
				Absent: true,
				Uid:    &uid,
			},
			ErrorContains: "'uid' can not be set with 'absent'",
		},
		{
			Title: "absent with user",
			File: File{
				Path:   "/foo",
				Absent: true,
				User:   &user,
			},
			ErrorContains: "'user' can not be set with 'absent'",
		},
		{
			Title: "absent with gid",
			File: File{
				Path:   "/foo",
				Absent: true,
				Gid:    &gid,
			},
			ErrorContains: "'gid' can not be set with 'absent'",
		},
		{
			Title: "absent with group",
			File: File{
				Path:   "/foo",
				Absent: true,
				Group:  &group,
			},
			ErrorContains: "'group' can not be set with 'absent'",
		},
		{
			Title: "absent with type definition",
			File: File{
				Path:         "/foo",
				Absent:       true,
				SymbolicLink: "/bar",
			},
			ErrorContains: "'symbolic_link' can not be set with 'absent'",
		},
		{
			Title: "multiple types",
			File: File{
				Path:   "/foo",
				FIFO:   true,
				Socket: true,
			},
			ErrorContains: "exactly one file type can be set: 'socket', 'symbolic_link', 'regular_file', 'block_device', 'directory', 'character_device' or 'fifo'",
		},
		// Socket
		{
			Title: "socket",
			File: File{
				Path:   "/foo",
				Socket: true,
			},
		},
		// SymbolicLink
		{
			Title: "symlink",
			File: File{
				Path:         "/foo",
				SymbolicLink: "/bar",
			},
		},
		{
			Title: "symlink with mode",
			File: File{
				Path:         "/foo",
				SymbolicLink: "/bar",
				Mode:         &mode,
			},
			ErrorContains: "'mode' can not be set with 'symbolic_link'",
		},
		// RegularFile
		{
			Title: "regular file",
			File: File{
				Path:        "/foo",
				RegularFile: new(string),
			},
		},
		// BlockDevice
		{
			Title: "block device",
			File: File{
				Path:        "/foo",
				BlockDevice: &device,
			},
		},
		// Directory
		{
			Title: "directory",
			File: File{
				Path: "/foo",
				Directory: &[]*File{
					{
						Path:        "/foo/bar",
						RegularFile: new(string),
					},
				},
			},
		},
		{
			Title: "directory entry not subpath",
			File: File{
				Path: "/foo",
				Directory: &[]*File{
					{
						Path:        "/fooz/bar",
						RegularFile: new(string),
					},
				},
			},
			ErrorContains: "is not a subpath",
		},
		// CharacterDevice
		{
			Title: "char device",
			File: File{
				Path:            "/foo",
				CharacterDevice: &device,
			},
		},
		// FIFO
		{
			Title: "fifo",
			File: File{
				Path: "/foo",
				FIFO: true,
			},
		},
		// Mode
		{
			Title: "invalid mode",
			File: File{
				Path:        "/foo",
				RegularFile: new(string),
				Mode:        &badMode,
			},
			ErrorContains: fmt.Sprintf("'mode' does not match mask 07777: %#o", badMode),
		},
		// Uid / User
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
			ErrorContains: "either 'user' or 'uid' can be set",
		},
		// Gid / Group
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
			ErrorContains: "either 'group' or 'gid' can be set",
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
}

func TestFile_Apply(t *testing.T) {
	ctx := t.Context()
	ctx = log.WithTestLogger(ctx)
	hst := host.Local{}
	var uid uint32 = uint32(os.Getuid())
	var gid uint32 = uint32(os.Getgid())
	var mode types.FileMode = 01724
	contents := "foo\nbar"
	var device types.FileDevice = 234122345

	testApplyFn := func(prefix string, expectedErr error) {
		socketPath := filepath.Join(prefix, "socket")
		symlinkPath := filepath.Join(prefix, "symlink")
		symlinkOldPath := "target"
		regularPath := filepath.Join(prefix, "regular")
		blockPath := filepath.Join(prefix, "block")
		characterPath := filepath.Join(prefix, "character")
		fifoPath := filepath.Join(prefix, "fifo")

		file := File{
			Path: prefix,
			Directory: &[]*File{
				{
					Path: fifoPath,
					FIFO: true,
					Mode: &mode,
					Uid:  &uid,
					Gid:  &gid,
				},
				{
					Path:        regularPath,
					RegularFile: &contents,
					Mode:        &mode,
					Uid:         &uid,
					Gid:         &gid,
				},
				{
					Path:   socketPath,
					Socket: true,
					Mode:   &mode,
					Uid:    &uid,
					Gid:    &gid,
				},
				{
					Path:         symlinkPath,
					SymbolicLink: symlinkOldPath,
					Uid:          &uid,
					Gid:          &gid,
				},
			},
			Mode: &mode,
			Uid:  &uid,
			Gid:  &gid,
		}

		if isRoot(t) {
			*file.Directory = append(
				[]*File{
					{
						Path:        blockPath,
						BlockDevice: &device,
						Mode:        &mode,
						Uid:         &uid,
						Gid:         &gid,
					},
					{
						Path:            characterPath,
						CharacterDevice: &device,
						Mode:            &mode,
						Uid:             &uid,
						Gid:             &gid,
					},
				},
				*file.Directory...,
			)
		}

		err := file.Apply(ctx, hst)
		if expectedErr == nil {
			require.NoError(t, err)

			loadedFile, err := LoadFile(ctx, hst, prefix)
			require.NoError(t, err)

			require.True(t, reflect.DeepEqual(file, loadedFile))
		} else {
			require.ErrorIs(t, err, expectedErr)
		}
	}
	t.Run("initial condition: absent", func(t *testing.T) {
		prefix := t.TempDir()
		require.NoError(t, os.Remove(prefix))
		testApplyFn(prefix, nil)
	})
	t.Run("initial condition: empty dir", func(t *testing.T) {
		prefix := t.TempDir()
		testApplyFn(prefix, nil)
	})
	t.Run("initial condition: different type", func(t *testing.T) {
		prefix := t.TempDir()
		require.NoError(t, os.Remove(prefix))
		file, err := os.Create(prefix)
		require.NoError(t, err)
		require.NoError(t, file.Close())
		testApplyFn(prefix, nil)
	})
	t.Run("initial condition: extra files in dir", func(t *testing.T) {
		prefix := t.TempDir()

		extraDirPath := filepath.Join(prefix, "extra_dir")
		require.NoError(t, os.Mkdir(extraDirPath, os.FileMode(0755)))

		extraFilePath := filepath.Join(extraDirPath, "extra_file")
		file, err := os.Create(extraFilePath)
		require.NoError(t, err)
		require.NoError(t, file.Close())

		testApplyFn(prefix, nil)
	})
	t.Run("initial condition: user has no permission to write", func(t *testing.T) {
		prefix := t.TempDir()
		require.NoError(t, os.Remove(prefix))
		file, err := os.Create(prefix)
		require.NoError(t, err)
		require.NoError(t, file.Close())
		require.NoError(t, os.Chmod(prefix, os.FileMode(0500)))
		testApplyFn(prefix, nil)
	})
}
