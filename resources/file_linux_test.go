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
	// TODO
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
