package resources

import (
	"context"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/resonance/diff"
	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
)

type FileTestCase struct {
	Title         string
	File          File
	ErrorContains string
}

func (f *FileTestCase) Run(t *testing.T) {
	t.Run(f.Title, func(t *testing.T) {
		err := f.File.Validate()
		if f.ErrorContains != "" {
			require.ErrorContains(t, err, f.ErrorContains)
		} else {
			require.NoError(t, err)
		}
	})
}

func isRoot(t *testing.T) bool {
	u, err := user.Current()
	require.NoError(t, err)
	return u.Uid == "0"
}

func TestFile(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)
	hst := host.Local{}

	user := "user"
	var uid uint32 = uint32(os.Getuid())
	group := "group"
	var gid uint32 = uint32(os.Getgid())
	var mode types.FileMode = 01724
	var badMode types.FileMode = 07777 + 1
	contents := "foo\nbar"
	var device types.FileDevice = 234122345

	t.Run("Validate()", func(t *testing.T) {
		for _, tc := range []FileTestCase{
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
				ErrorContains: "'path' must be absolute",
			},
			{
				Title: "not clean path",
				File: File{
					Path:   "//foo",
					Absent: true,
				},
				ErrorContains: "'path' must be clean",
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
				ErrorContains: "can not set 'mode' with absent",
			},
			{
				Title: "absent with uid",
				File: File{
					Path:   "/foo",
					Absent: true,
					Uid:    &uid,
				},
				ErrorContains: "can not set 'uid' with absent",
			},
			{
				Title: "absent with user",
				File: File{
					Path:   "/foo",
					Absent: true,
					User:   &user,
				},
				ErrorContains: "can not set 'user' with absent",
			},
			{
				Title: "absent with gid",
				File: File{
					Path:   "/foo",
					Absent: true,
					Gid:    &gid,
				},
				ErrorContains: "can not set 'gid' with absent",
			},
			{
				Title: "absent with group",
				File: File{
					Path:   "/foo",
					Absent: true,
					Group:  &group,
				},
				ErrorContains: "can not set 'group' with absent",
			},
			{
				Title: "absent with type definition",
				File: File{
					Path:         "/foo",
					Absent:       true,
					SymbolicLink: "/bar",
				},
				ErrorContains: "can not set 'absent' and a file type at the same time",
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
				ErrorContains: "can not set 'mode' with symlink",
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
					Directory: &[]File{
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
					Directory: &[]File{
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
				ErrorContains: "file mode does not match mask",
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
				ErrorContains: "can't set both 'uid' and 'user'",
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
				ErrorContains: "can't set both 'gid' and 'group'",
			},
		} {
			tc.Run(t)
		}
	})

	t.Run("Load()", func(t *testing.T) {
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

			file := File{Path: prefix}
			require.NoError(t, file.Load(ctx, hst))

			expectedFile := File{
				Path: prefix,
				Directory: &[]File{
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
					[]File{
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

			require.Truef(
				t,
				reflect.DeepEqual(expectedFile, file),
				diff.DiffAsYaml(expectedFile, file).String(),
			)
		})
		t.Run("absent", func(t *testing.T) {
			prefix := t.TempDir()
			path := filepath.Join(prefix, "absent")
			file := File{Path: path}
			err := file.Load(ctx, hst)
			require.NoError(t, err)
			require.True(t, reflect.DeepEqual(file, File{
				Path:   path,
				Absent: true,
			}))
		})
	})

	t.Run("Resolve()", func(t *testing.T) {
		path := "/foo"
		var uid uint32 = 0
		var defaultUid uint32 = 0
		var gid uint32 = 0
		var defaultGid uint32 = 0
		t.Run("User", func(t *testing.T) {
			user := "root"
			t.Run("valid", func(t *testing.T) {
				file := File{
					Path: path,
					User: &user,
				}
				require.NoError(t, file.Resolve(ctx, hst))
				expectedFile := File{
					Path: path,
					Uid:  &uid,
					Gid:  &defaultGid,
				}
				require.Truef(
					t,
					reflect.DeepEqual(expectedFile, file),
					diff.DiffAsYaml(expectedFile, file).String(),
				)
			})
			t.Run("invalid", func(t *testing.T) {
				badUser := "foobar"
				file := File{
					Path: path,
					User: &badUser,
				}
				require.ErrorContains(t, file.Resolve(ctx, hst), "unknown user")
			})
		})
		t.Run("Group", func(t *testing.T) {
			group := "root"
			t.Run("valid", func(t *testing.T) {
				file := File{
					Path:  path,
					Group: &group,
				}
				require.NoError(t, file.Resolve(ctx, hst))
				expectedFile := File{
					Path: path,
					Uid:  &defaultUid,
					Gid:  &gid,
				}
				require.Truef(
					t,
					reflect.DeepEqual(expectedFile, file),
					diff.DiffAsYaml(expectedFile, file).String(),
				)
			})
			t.Run("invalid", func(t *testing.T) {
				badGroup := "foobar"
				file := File{
					Path:  path,
					Group: &badGroup,
				}
				require.ErrorContains(t, file.Resolve(ctx, hst), "unknown group")
			})
		})
		t.Run("Directory", func(t *testing.T) {
			t.Run("sort & recursion", func(t *testing.T) {
				file := File{
					Path: path,
					Directory: &[]File{
						{
							Path:        filepath.Join(path, "last"),
							RegularFile: new(string),
						},
						{
							Path:        filepath.Join(path, "first"),
							RegularFile: new(string),
						},
					},
				}
				require.NoError(t, file.Resolve(ctx, hst))
				expectedFile := File{
					Path: path,
					Directory: &[]File{
						{
							Path:        filepath.Join(path, "first"),
							RegularFile: new(string),
							Uid:         &defaultUid,
							Gid:         &defaultGid,
						},
						{
							Path:        filepath.Join(path, "last"),
							RegularFile: new(string),
							Uid:         &defaultUid,
							Gid:         &defaultGid,
						},
					},
					Uid: &defaultUid,
					Gid: &defaultGid,
				}
				require.Truef(
					t,
					reflect.DeepEqual(expectedFile, file),
					diff.DiffAsYaml(expectedFile, file).String(),
				)
			})
		})
	})

	t.Run("Apply()", func(t *testing.T) {
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
				Directory: &[]File{
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
					[]File{
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

				loadedFile := File{Path: prefix}
				require.NoError(t, loadedFile.Load(ctx, hst))

				require.Truef(
					t,
					reflect.DeepEqual(file, loadedFile),
					diff.DiffAsYaml(file, loadedFile).String(),
				)
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
	})
}
