package host

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

func skipIfRoot(t *testing.T) {
	u, err := user.Current()
	require.NoError(t, err)
	if u.Uid == "0" {
		t.SkipNow()
	}
}

func testHost(t *testing.T, hst host.Host) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	var outputBuffer bytes.Buffer

	t.Run("Chmod", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "foo")
		file, err := os.Create(name)
		require.NoError(t, err)
		file.Close()
		t.Run("Success", func(t *testing.T) {
			outputBuffer.Reset()
			var fileMode uint32 = 03247
			err = hst.Chmod(ctx, name, fileMode)
			require.NoError(t, err)
			var stat_t syscall.Stat_t
			err := syscall.Lstat(name, &stat_t)
			require.NoError(t, err)
			require.Equal(t, fileMode, stat_t.Mode&07777)
		})
		t.Run("ErrPermission", func(t *testing.T) {
			outputBuffer.Reset()
			skipIfRoot(t)
			err = hst.Chmod(ctx, "/tmp", 0)
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			outputBuffer.Reset()
			err = hst.Chmod(ctx, "/non-existent", 0)
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
			err = hst.Chown(ctx, name, int(stat_t.Uid), int(stat_t.Gid))
			require.NoError(t, err)
		})
		t.Run("ErrPermission", func(t *testing.T) {
			outputBuffer.Reset()
			skipIfRoot(t)
			err = hst.Chown(ctx, name, 0, 0)
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			outputBuffer.Reset()
			err = hst.Chown(ctx, "/non-existent", 0, 0)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("Lookup", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			outputBuffer.Reset()
			u, err := hst.Lookup(ctx, "root")
			require.NoError(t, err)
			require.Equal(t, "0", u.Uid)
			require.Equal(t, "0", u.Gid)
			require.Equal(t, "root", u.Username)
		})
		t.Run("UnknownUserError", func(t *testing.T) {
			outputBuffer.Reset()
			_, err := hst.Lookup(ctx, "foobar")
			require.ErrorIs(t, err, user.UnknownUserError("foobar"))
		})
	})

	t.Run("LookupGroup", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			outputBuffer.Reset()
			g, err := hst.LookupGroup(ctx, "root")
			require.NoError(t, err)
			require.Equal(t, "0", g.Gid)
			require.Equal(t, "root", g.Name)
		})
		t.Run("UnknownGroupError", func(t *testing.T) {
			outputBuffer.Reset()
			_, err := hst.LookupGroup(ctx, "foobar")
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
			panic("TODO refactor to cover all file types & mode bits etc")
			// panic("TODO add test to see if Lstat does NOT follow symlink")
			// fileInfo, err := os.Lstat(name)
			// require.NoError(t, err)
			// _, err := hst.Lstat(ctx, name)

		})
		t.Run("ErrPermission", func(t *testing.T) {
			outputBuffer.Reset()
			_, err := hst.Lstat(ctx, "/etc/ssl/private/foo")
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			outputBuffer.Reset()
			_, err := hst.Lstat(ctx, "/non-existent")
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("Mkdir", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			outputBuffer.Reset()
			name := filepath.Join(t.TempDir(), "foo")
			fileMode := os.FileMode(0500)
			err := hst.Mkdir(ctx, name, fileMode)
			require.NoError(t, err)
			fileInfo, err := os.Lstat(name)
			require.NoError(t, err)
			require.True(t, fileInfo.IsDir())
			require.Equal(t, fileMode, fileInfo.Mode()&fs.ModePerm)
		})
		t.Run("ErrPermission", func(t *testing.T) {
			outputBuffer.Reset()
			skipIfRoot(t)
			err := hst.Mkdir(ctx, "/etc/foo", 0750)
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrExist", func(t *testing.T) {
			outputBuffer.Reset()
			name := filepath.Join(t.TempDir(), "foo")
			err := hst.Mkdir(ctx, name, 0750)
			require.NoError(t, err)
			err = hst.Mkdir(ctx, name, 0750)
			require.ErrorIs(t, err, os.ErrExist)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			outputBuffer.Reset()
			name := filepath.Join(t.TempDir(), "foo", "bar")
			err := hst.Mkdir(ctx, name, 0750)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("ReadFile", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			outputBuffer.Reset()
			name := filepath.Join(t.TempDir(), "foo")
			data := []byte("foo")
			err := os.WriteFile(name, data, os.FileMode(0600))
			require.NoError(t, err)
			readData, err := hst.ReadFile(ctx, name)
			require.NoError(t, err)
			require.Equal(t, data, readData)
		})
		t.Run("ErrPermission", func(t *testing.T) {
			outputBuffer.Reset()
			_, err := hst.ReadFile(ctx, "/etc/shadow")
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			outputBuffer.Reset()
			_, err := hst.ReadFile(ctx, "/non-existent")
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("Remove", func(t *testing.T) {
		t.Run("Success file", func(t *testing.T) {
			outputBuffer.Reset()
			name := filepath.Join(t.TempDir(), "foo")
			file, err := os.Create(name)
			require.NoError(t, err)
			file.Close()
			err = hst.Remove(ctx, name)
			require.NoError(t, err)
			_, err = os.Lstat(name)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
		t.Run("Success dir", func(t *testing.T) {
			outputBuffer.Reset()
			name := filepath.Join(t.TempDir(), "foo")
			err := os.Mkdir(name, 0700)
			require.NoError(t, err)
			err = hst.Remove(ctx, name)
			require.NoError(t, err)
			_, err = os.Lstat(name)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
		t.Run("ErrPermission file", func(t *testing.T) {
			outputBuffer.Reset()
			skipIfRoot(t)
			err := hst.Remove(ctx, "/bin/ls")
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrPermission dir", func(t *testing.T) {
			outputBuffer.Reset()
			skipIfRoot(t)
			err := hst.Remove(ctx, "/bin")
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			outputBuffer.Reset()
			err := hst.Remove(ctx, "/non-existent")
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("Args, output and failure", func(t *testing.T) {
			outputBuffer.Reset()
			waitStatus, stdout, stderr, err := host.Run(ctx, hst, host.Cmd{
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
				waitStatus, stdout, stderr, err := host.Run(ctx, hst, host.Cmd{
					Path: "env",
				})
				t.Cleanup(func() {
					if t.Failed() {
						t.Logf("output:\n%s\n", outputBuffer.String())
						t.Logf("stdout:\n%s\nstderr:\n%s\n", stdout, stderr)
					}
				})
				var envPath string
				for _, value := range os.Environ() {
					if strings.HasPrefix(value, "PATH=") {
						envPath = value
						break
					}
				}
				require.True(t, strings.HasPrefix(envPath, "PATH="))
				require.NoError(t, err)
				require.True(t, waitStatus.Success())
				require.Equal(t, 0, waitStatus.ExitCode)
				require.True(t, waitStatus.Exited)
				require.Equal(t, "", waitStatus.Signal)
				expectedEnv := []string{
					"LANG=en_US.UTF-8",
					envPath,
				}
				sort.Strings(expectedEnv)
				receivedEnv := []string{}
				for _, value := range strings.Split(stdout, "\n") {
					if value == "" {
						continue
					}
					receivedEnv = append(receivedEnv, value)
				}
				sort.Strings(receivedEnv)
				require.Equal(t, expectedEnv, receivedEnv)
				require.Empty(t, stderr)
			})
			t.Run("Set", func(t *testing.T) {
				outputBuffer.Reset()
				env := "FOO=bar"
				waitStatus, stdout, stderr, err := host.Run(ctx, hst, host.Cmd{
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
			dir := t.TempDir()
			waitStatus, stdout, stderr, err := host.Run(ctx, hst, host.Cmd{
				Path: "pwd",
				Dir:  dir,
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
			require.Equal(t, fmt.Sprintf("%s\n", dir), stdout)
			require.Equal(t, "", stderr)
		})
		t.Run("Stdin", func(t *testing.T) {
			outputBuffer.Reset()
			stdin := "hello"
			waitStatus, stdout, stderr, err := host.Run(ctx, hst, host.Cmd{
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

	t.Run("WriteFile", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			outputBuffer.Reset()
			name := filepath.Join(t.TempDir(), "foo")
			data := []byte("foo")
			fileMode := os.FileMode(0600)
			err := hst.WriteFile(ctx, name, data, fileMode)
			require.NoError(t, err)
			readData, err := os.ReadFile(name)
			require.NoError(t, err)
			require.Equal(t, data, readData)
			fileInfo, err := os.Lstat(name)
			require.NoError(t, err)
			require.False(t, fileInfo.IsDir())
			require.Equal(t, fileMode, fileInfo.Mode()&fs.ModePerm)
		})
		t.Run("ErrPermission", func(t *testing.T) {
			outputBuffer.Reset()
			skipIfRoot(t)
			err := hst.WriteFile(ctx, "/etc/foo", []byte{}, 0600)
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ovewrite file", func(t *testing.T) {
			outputBuffer.Reset()
			name := filepath.Join(t.TempDir(), "foo")
			fileMode := os.FileMode(0600)
			err := hst.WriteFile(ctx, name, []byte{}, fileMode)
			require.NoError(t, err)
			data := []byte("foo")
			newFileMode := os.FileMode(0640)
			err = hst.WriteFile(ctx, name, data, newFileMode)
			require.NoError(t, err)
			readData, err := os.ReadFile(name)
			require.NoError(t, err)
			require.Equal(t, data, readData)
			fileInfo, err := os.Lstat(name)
			require.NoError(t, err)
			require.False(t, fileInfo.IsDir())
			require.Equal(t, fileMode, fileInfo.Mode()&fs.ModePerm)
		})
		t.Run("is directory", func(t *testing.T) {
			outputBuffer.Reset()
			name := filepath.Join(t.TempDir(), "foo")
			err := os.Mkdir(name, os.FileMode(0700))
			require.NoError(t, err)
			err = hst.WriteFile(ctx, name, []byte{}, os.FileMode(0640))
			require.Error(t, err)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			outputBuffer.Reset()
			name := filepath.Join(t.TempDir(), "foo", "bar")
			err := hst.WriteFile(ctx, name, []byte{}, 0600)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})
}
