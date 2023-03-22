package host

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/log"
)

func checkNotRoot(t *testing.T) {
	u, err := user.Current()
	require.NoError(t, err)
	require.NotEqual(t, "0", u.Uid, "test can not be executed as root")
}

func testHost(t *testing.T, host Host) {
	ctx := context.Background()
	var outputBuffer bytes.Buffer
	ctx = log.SetLoggerValue(ctx, &outputBuffer, "debug", func(code int) {
		t.Fatalf("exit called with %d", code)
	})

	t.Run("Chmod", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "foo")
		file, err := os.Create(name)
		require.NoError(t, err)
		file.Close()
		t.Run("Success", func(t *testing.T) {
			outputBuffer.Reset()
			fileMode := os.FileMode(0247)
			err = host.Chmod(ctx, name, fileMode)
			require.NoError(t, err)
			fileInfo, err := os.Lstat(name)
			require.NoError(t, err)
			require.Equal(t, fileMode, fileInfo.Mode())
		})
		t.Run("ErrPermission", func(t *testing.T) {
			outputBuffer.Reset()
			checkNotRoot(t)
			err = host.Chmod(ctx, "/tmp", 0)
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			outputBuffer.Reset()
			err = host.Chmod(ctx, "/non-existent", 0)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("Chown", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "foo")
		file, err := os.Create(name)
		require.NoError(t, err)
		file.Close()
		t.Run("Success", func(t *testing.T) {
			outputBuffer.Reset()
			fileInfo, err := os.Lstat(name)
			require.NoError(t, err)
			stat_t := fileInfo.Sys().(*syscall.Stat_t)
			err = host.Chown(ctx, name, int(stat_t.Uid), int(stat_t.Gid))
			require.NoError(t, err)
		})
		t.Run("ErrPermission", func(t *testing.T) {
			outputBuffer.Reset()
			checkNotRoot(t)
			err = host.Chown(ctx, name, 0, 0)
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			outputBuffer.Reset()
			err = host.Chown(ctx, "/non-existent", 0, 0)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("Lookup", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			outputBuffer.Reset()
			u, err := host.Lookup(ctx, "root")
			require.NoError(t, err)
			require.Equal(t, "0", u.Uid)
			require.Equal(t, "0", u.Gid)
			require.Equal(t, "root", u.Username)
		})
		t.Run("UnknownUserError", func(t *testing.T) {
			outputBuffer.Reset()
			_, err := host.Lookup(ctx, "foobar")
			require.ErrorIs(t, err, user.UnknownUserError("foobar"))
		})
	})

	t.Run("LookupGroup", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			outputBuffer.Reset()
			g, err := host.LookupGroup(ctx, "root")
			require.NoError(t, err)
			require.Equal(t, "0", g.Gid)
			require.Equal(t, "root", g.Name)
		})
		t.Run("UnknownGroupError", func(t *testing.T) {
			outputBuffer.Reset()
			_, err := host.LookupGroup(ctx, "foobar")
			require.ErrorIs(t, err, user.UnknownGroupError("foobar"))
		})
	})

	t.Run("Lstat", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "foo")
		file, err := os.Create(name)
		require.NoError(t, err)
		file.Close()
		t.Run("Success", func(t *testing.T) {
			outputBuffer.Reset()
			fileInfo, err := os.Lstat(name)
			require.NoError(t, err)
			hostFileInfo, err := host.Lstat(ctx, name)
			require.NoError(t, err)
			require.Equal(t, fileInfo, hostFileInfo)
		})
		t.Run("ErrPermission", func(t *testing.T) {
			outputBuffer.Reset()
			_, err := host.Lstat(ctx, "/etc/ssl/private/foo")
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			outputBuffer.Reset()
			_, err := host.Lstat(ctx, "/non-existent")
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("Mkdir", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			outputBuffer.Reset()
			name := filepath.Join(t.TempDir(), "foo")
			fileMode := os.FileMode(0500)
			err := host.Mkdir(ctx, name, fileMode)
			require.NoError(t, err)
			fileInfo, err := os.Lstat(name)
			require.NoError(t, err)
			require.True(t, fileInfo.IsDir())
			require.Equal(t, fileMode, fileInfo.Mode()&fs.ModePerm)
		})
		t.Run("ErrPermission", func(t *testing.T) {
			outputBuffer.Reset()
			checkNotRoot(t)
			err := host.Mkdir(ctx, "/etc/foo", 0750)
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrExist", func(t *testing.T) {
			outputBuffer.Reset()
			name := filepath.Join(t.TempDir(), "foo")
			err := host.Mkdir(ctx, name, 0750)
			require.NoError(t, err)
			err = host.Mkdir(ctx, name, 0750)
			require.ErrorIs(t, err, os.ErrExist)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			outputBuffer.Reset()
		})
	})

	// t.Run("ReadFile", func(t *testing.T) {
	// 	t.Run("Success", func(t *testing.T) {
	// outputBuffer.Reset()
	// 	})
	// 	t.Run("ErrPermission", func(t *testing.T) {
	// outputBuffer.Reset()
	// 	})
	// 	t.Run("ErrExist", func(t *testing.T) {
	// outputBuffer.Reset()
	// 	})
	// 	t.Run("ErrNotExist", func(t *testing.T) {
	// outputBuffer.Reset()
	// 	})
	// })

	// t.Run("Remove", func(t *testing.T) {
	// 	t.Run("Success", func(t *testing.T) {
	// outputBuffer.Reset()
	// 	})
	// 	t.Run("ErrPermission", func(t *testing.T) {
	// outputBuffer.Reset()
	// 	})
	// 	t.Run("ErrExist", func(t *testing.T) {
	// outputBuffer.Reset()
	// 	})
	// 	t.Run("ErrNotExist", func(t *testing.T) {
	// outputBuffer.Reset()
	// 	})
	// })

	t.Run("Run", func(t *testing.T) {
		t.Run("Args, output and failure", func(t *testing.T) {
			outputBuffer.Reset()
			waitStatus, stdout, stderr, err := host.Run(ctx, Cmd{
				Path: "ls",
				Args: []string{"-d", "../tmp", "/non-existent"},
			})
			require.NoError(t, err)
			t.Cleanup(func() {
				if t.Failed() {
					t.Logf("\noutput:\n%s\n", outputBuffer.String())
					t.Logf("\nstdout:\n%s\nstderr:\n%s\n", stdout, stderr)
				}
			})
			require.False(t, waitStatus.Success())
			require.Equal(t, 2, waitStatus.ExitCode)
			require.True(t, waitStatus.Exited)
			require.Equal(t, "", waitStatus.Signal)
			require.Equal(t, "../tmp\n", stdout)
			require.Equal(t, "ls: cannot access '/non-existent': No such file or directory\n", stderr)
		})
		t.Run("Env", func(t *testing.T) {
			t.Run("Empty", func(t *testing.T) {
				outputBuffer.Reset()
				waitStatus, stdout, stderr, err := host.Run(ctx, Cmd{
					Path: "env",
				})
				require.NoError(t, err)
				require.True(t, waitStatus.Success())
				require.Equal(t, 0, waitStatus.ExitCode)
				require.True(t, waitStatus.Exited)
				require.Equal(t, "", waitStatus.Signal)
				require.Equal(t, "LANG=en_US.UTF-8\n", stdout)
				require.Empty(t, stderr)
			})
			t.Run("Set", func(t *testing.T) {
				outputBuffer.Reset()
				env := "FOO=bar"
				waitStatus, stdout, stderr, err := host.Run(ctx, Cmd{
					Path: "env",
					Env:  []string{env},
				})
				require.NoError(t, err)
				t.Cleanup(func() {
					if t.Failed() {
						t.Logf("output:\n%s\n", outputBuffer.String())
						t.Logf("stdout:\n%s\nstderr:\n%s\n", stdout, stderr)
					}
				})
				require.True(t, waitStatus.Success())
				require.Equal(t, 0, waitStatus.ExitCode)
				require.True(t, waitStatus.Exited)
				require.Equal(t, "", waitStatus.Signal)
				require.Equal(t, env, strings.TrimRight(stdout, "\n"))
				require.Empty(t, stderr)
			})
		})
		t.Run("Dir", func(t *testing.T) {
			outputBuffer.Reset()
			wd, err := os.Getwd()
			require.NoError(t, err)
			waitStatus, stdout, stderr, err := host.Run(ctx, Cmd{
				Path: "pwd",
				Dir:  wd,
			})
			require.NoError(t, err)
			t.Cleanup(func() {
				if t.Failed() {
					t.Logf("\noutput:\n%s\n", outputBuffer.String())
					t.Logf("\nstdout:\n%s\nstderr:\n%s\n", stdout, stderr)
				}
			})
			require.True(t, waitStatus.Success())
			require.Equal(t, 0, waitStatus.ExitCode)
			require.True(t, waitStatus.Exited)
			require.Equal(t, "", waitStatus.Signal)
			require.Equal(t, fmt.Sprintf("%s\n", wd), stdout)
			require.Equal(t, "", stderr)
		})
		t.Run("Stdin", func(t *testing.T) {
			outputBuffer.Reset()
			stdin := "hello"
			waitStatus, stdout, stderr, err := host.Run(ctx, Cmd{
				Path:  "sh",
				Args:  []string{"-c", "read v && echo =$v="},
				Stdin: strings.NewReader(fmt.Sprintf("%s\n", stdin)),
			})
			require.NoError(t, err)
			t.Cleanup(func() {
				if t.Failed() {
					t.Logf("\noutput:\n%s\n", outputBuffer.String())
					t.Logf("\nstdout:\n%s\nstderr:\n%s\n", stdout, stderr)
				}
			})
			require.True(t, waitStatus.Success())
			require.Equal(t, 0, waitStatus.ExitCode)
			require.True(t, waitStatus.Exited)
			require.Equal(t, "", waitStatus.Signal)
			require.Equal(t, fmt.Sprintf("=%s=\n", stdin), stdout)
			require.Equal(t, "", stderr)
		})
	})

	// t.Run("WriteFile", func(t *testing.T) {
	// 	t.Run("Success", func(t *testing.T) {
	// outputBuffer.Reset()
	// 	})
	// 	t.Run("ErrPermission", func(t *testing.T) {
	// outputBuffer.Reset()
	// 	})
	// 	t.Run("ErrExist", func(t *testing.T) {
	// outputBuffer.Reset()
	// 	})
	// 	t.Run("ErrNotExist", func(t *testing.T) {
	// outputBuffer.Reset()
	// 	})
	// })
}
