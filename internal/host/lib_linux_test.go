package host

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host"
)

func isRoot(t *testing.T) bool {
	u, err := user.Current()
	require.NoError(t, err)
	return u.Uid == "0"
}

func skipIfRoot(t *testing.T) {
	if isRoot(t) {
		t.SkipNow()
	}
}

func getBlockDevicePath(t *testing.T) string {
	dirEntries, err := os.ReadDir("/dev")
	require.NoError(t, err)
	for _, dirEntry := range dirEntries {
		fileInfo, err := dirEntry.Info()
		require.NoError(t, err)
		stat_t := fileInfo.Sys().(*syscall.Stat_t)
		if (stat_t.Mode & syscall.S_IFMT) == syscall.S_IFBLK {
			return filepath.Join("/dev", dirEntry.Name())
		}
	}
	t.SkipNow()
	return ""
}

var allModeBits = []uint32{
	syscall.S_ISUID, // 04000 set-user-ID bit
	syscall.S_ISGID, // 02000 set-group-ID bit
	syscall.S_ISVTX, // 01000 sticky bit
	syscall.S_IRUSR, // 00400 owner has read permission
	syscall.S_IWUSR, // 00200 owner has write permission
	syscall.S_IXUSR, // 00100 owner has execute permission
	syscall.S_IRGRP, // 00040 group has read permission
	syscall.S_IWGRP, // 00020 group has write permission
	syscall.S_IXGRP, // 00010 group has execute permission
	syscall.S_IROTH, // 00004 others have read permission
	syscall.S_IWOTH, // 00002 others have write permission
	syscall.S_IXOTH, // 00001 others have execute permission
}

func testBaseHost(
	t *testing.T,
	ctx context.Context,
	baseHost host.BaseHost,
	baseHostString,
	baseHostType string,
) {
	t.Run("Run", func(t *testing.T) {
		t.Run("Args, output and failure", func(t *testing.T) {
			var outputBuffer bytes.Buffer
			waitStatus, stdout, stderr, err := host.Run(ctx, baseHost, host.Cmd{
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
				var outputBuffer bytes.Buffer
				waitStatus, stdout, stderr, err := host.Run(ctx, baseHost, host.Cmd{
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
				var outputBuffer bytes.Buffer
				env := "FOO=bar"
				waitStatus, stdout, stderr, err := host.Run(ctx, baseHost, host.Cmd{
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
			t.Run("Success", func(t *testing.T) {
				var outputBuffer bytes.Buffer
				dir := t.TempDir()
				waitStatus, stdout, stderr, err := host.Run(ctx, baseHost, host.Cmd{
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
			t.Run("path must be absolute", func(t *testing.T) {
				_, _, _, err := host.Run(ctx, baseHost, host.Cmd{
					Path: "pwd",
					Dir:  "foo/bar",
				})
				require.ErrorContains(t, err, "path must be absolute")
			})
		})
		t.Run("Stdin", func(t *testing.T) {
			var outputBuffer bytes.Buffer
			stdin := "hello"
			waitStatus, stdout, stderr, err := host.Run(ctx, baseHost, host.Cmd{
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

	t.Run("String()", func(t *testing.T) {
		require.Equal(t, baseHostString, baseHost.String())
	})

	t.Run("Type()", func(t *testing.T) {
		require.Equal(t, baseHostType, baseHost.Type())
	})

	t.Run("Close()", func(t *testing.T) {
		t.SkipNow()
		require.NoError(t, baseHost.Close(ctx))
	})
}

func testHost(
	t *testing.T,
	ctx context.Context,
	hst host.Host,
	hostString,
	hostType string,
) {
	testBaseHost(t, ctx, hst, hostString, hostType)

	t.Run("Getuid", func(t *testing.T) {
		uid, err := hst.Geteuid(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(syscall.Getuid()), uid)
	})

	t.Run("Getgid", func(t *testing.T) {
		gid, err := hst.Getegid(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(syscall.Getgid()), gid)
	})

	t.Run("Chmod", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "foo")
		file, err := os.Create(name)
		require.NoError(t, err)
		file.Close()
		t.Run("Success", func(t *testing.T) {
			var fileMode uint32 = 01257
			err = hst.Chmod(ctx, name, fileMode)
			require.NoError(t, err)
			var stat_t syscall.Stat_t
			require.NoError(t, syscall.Lstat(name, &stat_t))
			require.Equal(t, fileMode, stat_t.Mode&07777)
		})
		t.Run("path must be absolute", func(t *testing.T) {
			err = hst.Chmod(ctx, "foo/bar", 0644)
			require.ErrorContains(t, err, "path must be absolute")
		})
		t.Run("ErrPermission", func(t *testing.T) {
			skipIfRoot(t)
			err = hst.Chmod(ctx, "/tmp", 0)
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
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
			fileInfo, err := os.Lstat(name)
			require.NoError(t, err)
			stat_t := fileInfo.Sys().(*syscall.Stat_t)
			err = hst.Chown(ctx, name, stat_t.Uid, stat_t.Gid)
			require.NoError(t, err)
		})
		t.Run("path must be absolute", func(t *testing.T) {
			err = hst.Chown(ctx, "foo/bar", 0, 0)
			require.ErrorContains(t, err, "path must be absolute")
		})
		t.Run("ErrPermission", func(t *testing.T) {
			skipIfRoot(t)
			err = hst.Chown(ctx, name, 0, 0)
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			err = hst.Chown(ctx, "/non-existent", 0, 0)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("Lookup", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			u, err := hst.Lookup(ctx, "root")
			require.NoError(t, err)
			require.Equal(t, "0", u.Uid)
			require.Equal(t, "0", u.Gid)
			require.Equal(t, "root", u.Username)
		})
		t.Run("UnknownUserError", func(t *testing.T) {
			_, err := hst.Lookup(ctx, "foobar")
			require.ErrorIs(t, err, user.UnknownUserError("foobar"))
		})
	})

	t.Run("LookupGroup", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			g, err := hst.LookupGroup(ctx, "root")
			require.NoError(t, err)
			require.Equal(t, "0", g.Gid)
			require.Equal(t, "root", g.Name)
		})
		t.Run("UnknownGroupError", func(t *testing.T) {
			_, err := hst.LookupGroup(ctx, "foobar")
			require.ErrorIs(t, err, user.UnknownGroupError("foobar"))
		})
	})

	t.Run("Lstat", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {

			name := filepath.Join(t.TempDir(), "foo")
			file, err := os.Create(name)
			require.NoError(t, err)
			file.Close()

			var expectedStat_t syscall.Stat_t
			err = syscall.Lstat(name, &expectedStat_t)
			require.NoError(t, err)

			stat_t, err := hst.Lstat(ctx, name)
			require.NoError(t, err)

			t.Run("Dev", func(t *testing.T) {
				require.Equal(t, expectedStat_t.Dev, stat_t.Dev)
			})
			t.Run("Ino", func(t *testing.T) {
				require.Equal(t, expectedStat_t.Ino, stat_t.Ino)
			})
			t.Run("Nlink", func(t *testing.T) {
				require.Equal(t, uint64(expectedStat_t.Nlink), stat_t.Nlink)
			})
			t.Run("Mode", func(t *testing.T) {
				t.Run("S_IFMT", func(t *testing.T) {
					t.Run("socket", func(t *testing.T) {
						socketPath := filepath.Join(t.TempDir(), "socket")
						listener, err := net.Listen("unix", socketPath)
						require.NoError(t, err)
						defer listener.Close()
						socketStat_t, err := hst.Lstat(ctx, socketPath)
						require.NoError(t, err)
						require.True(t, socketStat_t.Mode&syscall.S_IFMT == syscall.S_IFSOCK)
					})
					t.Run("symbolic link", func(t *testing.T) {
						linkPath := filepath.Join(t.TempDir(), "symlink")
						require.NoError(t, syscall.Symlink("foo", linkPath))
						linkStat_t, err := hst.Lstat(ctx, linkPath)
						require.NoError(t, err)
						require.True(t, linkStat_t.Mode&syscall.S_IFMT == syscall.S_IFLNK)
					})
					t.Run("regular file", func(t *testing.T) {
						require.True(t, stat_t.Mode&syscall.S_IFMT == syscall.S_IFREG)
					})
					t.Run("block device", func(t *testing.T) {
						blockStat_t, err := hst.Lstat(ctx, getBlockDevicePath(t))
						require.NoError(t, err)
						require.True(t, blockStat_t.Mode&syscall.S_IFMT == syscall.S_IFBLK)
					})
					t.Run("directory", func(t *testing.T) {
						dirStat_t, err := hst.Lstat(ctx, "/dev")
						require.NoError(t, err)
						require.True(t, dirStat_t.Mode&syscall.S_IFMT == syscall.S_IFDIR)
					})
					t.Run("character device", func(t *testing.T) {
						charStat_t, err := hst.Lstat(ctx, "/dev/tty")
						require.NoError(t, err)
						require.True(t, charStat_t.Mode&syscall.S_IFMT == syscall.S_IFCHR)
					})
					t.Run("FIFO", func(t *testing.T) {
						fifoPath := filepath.Join(t.TempDir(), "fifo")
						require.NoError(t, syscall.Mkfifo(fifoPath, 0644))
						fifoStat_t, err := hst.Lstat(ctx, fifoPath)
						require.NoError(t, err)
						require.True(t, fifoStat_t.Mode&syscall.S_IFMT == syscall.S_IFIFO)
					})
				})
				for _, modeBits := range allModeBits {
					t.Run(fmt.Sprintf("mode=%#.12o", modeBits), func(t *testing.T) {
						err := syscall.Chmod(name, modeBits)
						require.NoError(t, err)

						stat_t, err := hst.Lstat(ctx, name)
						require.NoError(t, err)

						require.Equal(t, modeBits, stat_t.Mode&07777)
					})
				}
			})
			t.Run("Uid", func(t *testing.T) {
				require.Equal(t, expectedStat_t.Uid, stat_t.Uid)
			})
			t.Run("Gid", func(t *testing.T) {
				require.Equal(t, expectedStat_t.Gid, stat_t.Gid)
			})
			t.Run("Rdev", func(t *testing.T) {
				require.Equal(t, expectedStat_t.Rdev, stat_t.Rdev)
			})
			t.Run("Size", func(t *testing.T) {
				require.Equal(t, expectedStat_t.Size, stat_t.Size)
			})
			t.Run("Blksize", func(t *testing.T) {
				// This value may differ (eg: stat syscall returns 512 while stat command returns 4k),
				// so we assert only that this is non-zero.
				require.Greater(t, stat_t.Blksize, int64(0))
			})
			t.Run("Blocks", func(t *testing.T) {
				require.Equal(t, expectedStat_t.Blocks, stat_t.Blocks)
			})
			t.Run("Atim", func(t *testing.T) {
				require.Equal(
					t,
					time.Unix(int64(expectedStat_t.Atim.Sec), int64(expectedStat_t.Atim.Nsec)),
					time.Unix(stat_t.Atim.Sec, stat_t.Atim.Nsec),
				)
			})
			t.Run("Mtim", func(t *testing.T) {
				require.Equal(
					t,
					time.Unix(int64(expectedStat_t.Mtim.Sec), int64(expectedStat_t.Mtim.Nsec)),
					time.Unix(stat_t.Mtim.Sec, stat_t.Mtim.Nsec),
				)
			})
			t.Run("Ctim", func(t *testing.T) {
				require.Equal(
					t,
					time.Unix(int64(expectedStat_t.Ctim.Sec), int64(expectedStat_t.Ctim.Nsec)),
					time.Unix(stat_t.Ctim.Sec, stat_t.Ctim.Nsec),
				)
			})
		})
		t.Run("path must be absolute", func(t *testing.T) {
			_, err := hst.Lstat(ctx, "foo/bar")
			require.ErrorContains(t, err, "path must be absolute")
		})
		t.Run("ErrPermission", func(t *testing.T) {
			skipIfRoot(t)
			_, err := hst.Lstat(ctx, "/etc/ssl/private/foo")
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			_, err := hst.Lstat(ctx, "/non-existent")
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("ReadDir", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {

			dirPath := t.TempDir()

			expectedTypeMap := map[string]uint8{}

			// socket
			socketPath := filepath.Join(dirPath, "socket")
			listener, err := net.Listen("unix", socketPath)
			require.NoError(t, err)
			defer listener.Close()
			expectedTypeMap["socket"] = syscall.DT_SOCK

			// symbolic link
			linkPath := filepath.Join(dirPath, "symlink")
			require.NoError(t, syscall.Symlink("foo", linkPath))
			expectedTypeMap["symlink"] = syscall.DT_LNK

			// regular file
			file, err := os.Create(filepath.Join(dirPath, "regular"))
			require.NoError(t, err)
			defer file.Close()
			expectedTypeMap["regular"] = syscall.DT_REG

			// block device: can't test without root

			// directory
			require.NoError(t, os.Mkdir(filepath.Join(dirPath, "directory"), os.FileMode(0700)))
			expectedTypeMap["directory"] = syscall.DT_DIR

			// character device: can't test without root

			// FIFO
			require.NoError(t, syscall.Mkfifo(filepath.Join(dirPath, "FIFO"), 0644))
			expectedTypeMap["FIFO"] = syscall.DT_FIFO

			dirEnts, err := hst.ReadDir(ctx, dirPath)
			require.NoError(t, err)

			inodeMap := map[uint64]bool{}
			for _, dirEnt := range dirEnts {
				require.NotContains(t, inodeMap, dirEnt.Ino)
				inodeMap[dirEnt.Ino] = true
				require.Contains(t, expectedTypeMap, dirEnt.Name)
				require.Equal(t, expectedTypeMap[dirEnt.Name], dirEnt.Type)
				delete(expectedTypeMap, dirEnt.Name)
			}
			require.Empty(t, expectedTypeMap)
		})
		t.Run("path must be absolute", func(t *testing.T) {
			_, err := hst.ReadDir(ctx, "foo")
			require.ErrorContains(t, err, "path must be absolute")
		})
		t.Run("ErrPermission", func(t *testing.T) {
			skipIfRoot(t)
			_, err := hst.ReadDir(ctx, "/etc/ssl/private/foo")
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			_, err := hst.ReadDir(ctx, "/non-existent")
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("Mkdir", func(t *testing.T) {
		for _, fileMode := range allModeBits {
			t.Run(fmt.Sprintf("Success mode=%#.12o", fileMode), func(t *testing.T) {
				name := filepath.Join(t.TempDir(), "foo")
				err := hst.Mkdir(ctx, name, fileMode)
				require.NoError(t, err)
				var stat_t syscall.Stat_t
				require.NoError(t, syscall.Lstat(name, &stat_t))
				require.True(t, stat_t.Mode&syscall.S_IFMT == syscall.S_IFDIR)
				require.Equal(t, fileMode, stat_t.Mode&07777)
			})
		}
		t.Run("path must be absolute", func(t *testing.T) {
			err := hst.Mkdir(ctx, "foo/bar", 0750)
			require.ErrorContains(t, err, "path must be absolute")
		})
		t.Run("ErrPermission", func(t *testing.T) {
			skipIfRoot(t)
			err := hst.Mkdir(ctx, "/etc/foo", 0750)
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrExist", func(t *testing.T) {
			name := filepath.Join(t.TempDir(), "foo")
			err := hst.Mkdir(ctx, name, 0750)
			require.NoError(t, err)
			err = hst.Mkdir(ctx, name, 0750)
			require.ErrorIs(t, err, os.ErrExist)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			name := filepath.Join(t.TempDir(), "foo", "bar")
			err := hst.Mkdir(ctx, name, 0750)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("ReadFile", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			name := filepath.Join(t.TempDir(), "foo")
			data := []byte("foo")
			err := os.WriteFile(name, data, os.FileMode(0600))
			require.NoError(t, err)
			fileReadCloser, err := hst.ReadFile(ctx, name)
			require.NoError(t, err)
			readData, err := io.ReadAll(fileReadCloser)
			assert.NoError(t, err)
			assert.Equal(t, data, readData)
			require.NoError(t, fileReadCloser.Close())
		})
		t.Run("path must be absolute", func(t *testing.T) {
			fileReadCloser, err := hst.ReadFile(ctx, "foo/bar")
			require.ErrorContains(t, err, "path must be absolute")
			if err == nil {
				require.NoError(t, fileReadCloser.Close())
			}
		})
		t.Run("ErrPermission", func(t *testing.T) {
			fileReadCloser, err := hst.ReadFile(ctx, "/etc/shadow")
			skipIfRoot(t)
			require.ErrorIs(t, err, os.ErrPermission)
			if err == nil {
				require.NoError(t, fileReadCloser.Close())
			}
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			fileReadCloser, err := hst.ReadFile(ctx, "/non-existent")
			require.ErrorIs(t, err, os.ErrNotExist)
			if err == nil {
				require.NoError(t, fileReadCloser.Close())
			}
		})
	})

	t.Run("Symlink", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			tempDir := t.TempDir()
			newname := filepath.Join(tempDir, "newname")
			oldname := "foo/bar"
			err := hst.Symlink(ctx, oldname, newname)
			require.NoError(t, err)
			readOldname, err := os.Readlink(newname)
			require.NoError(t, err)
			require.Equal(t, oldname, readOldname)
		})
		t.Run("path must be absolute", func(t *testing.T) {
			newname := "relative/path"
			oldname := "foo/bar"
			err := hst.Symlink(ctx, oldname, newname)
			require.ErrorContains(t, err, "path must be absolute")
		})
		t.Run("ErrPermission", func(t *testing.T) {
			skipIfRoot(t)
			newname := "/etc/foo"
			oldname := "foo/bar"
			err := hst.Symlink(ctx, oldname, newname)
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrExist", func(t *testing.T) {
			tempDir := t.TempDir()
			newname := filepath.Join(tempDir, "newname")
			oldname := "foo/bar"
			err := hst.Symlink(ctx, oldname, newname)
			require.NoError(t, err)
			err = hst.Symlink(ctx, oldname, newname)
			require.ErrorIs(t, err, os.ErrExist)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			oldname := "foo/bar"
			newname := "/bad/path"
			err := hst.Symlink(ctx, oldname, newname)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("Readlink", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			tempDir := t.TempDir()
			target := filepath.Join(tempDir, "target")
			err := os.WriteFile(target, []byte("content"), 0644)
			require.NoError(t, err)
			symlink := filepath.Join(tempDir, "symlink")
			err = os.Symlink(target, symlink)
			require.NoError(t, err)
			result, err := hst.Readlink(ctx, symlink)
			require.NoError(t, err)
			require.Equal(t, target, result)
		})
		t.Run("path must be absolute", func(t *testing.T) {
			_, err := hst.Readlink(ctx, "foo/bar")
			require.ErrorContains(t, err, "path must be absolute")
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			_, err := hst.Readlink(ctx, "/non-existent")
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("Remove", func(t *testing.T) {
		t.Run("Success file", func(t *testing.T) {
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
			name := filepath.Join(t.TempDir(), "foo")
			err := os.Mkdir(name, 0700)
			require.NoError(t, err)
			err = hst.Remove(ctx, name)
			require.NoError(t, err)
			_, err = os.Lstat(name)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
		t.Run("path must be absolute", func(t *testing.T) {
			err := hst.Remove(ctx, "foo/bar")
			require.ErrorContains(t, err, "path must be absolute")
		})
		t.Run("ErrPermission file", func(t *testing.T) {
			skipIfRoot(t)
			err := hst.Remove(ctx, "/bin/ls")
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrPermission dir", func(t *testing.T) {
			skipIfRoot(t)
			err := hst.Remove(ctx, "/bin")
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			err := hst.Remove(ctx, "/non-existent")
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})

	t.Run("Mknod", func(t *testing.T) {
		testMknod := func(t *testing.T, name string, fileType, modeBits uint32, dev uint64) {
			t.Run(name, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), name)
				var fileTypeBits uint32 = fileType
				var mode uint32 = fileTypeBits | modeBits

				var isDevice bool
				switch fileType & syscall.S_IFMT {
				case syscall.S_IFBLK, syscall.S_IFCHR:
					isDevice = true
				}

				err := hst.Mknod(ctx, path, mode, dev)
				if isDevice && !isRoot(t) {
					require.ErrorIs(t, err, os.ErrPermission)
					return
				}
				require.NoError(t, err)

				var stat_t syscall.Stat_t
				err = syscall.Stat(path, &stat_t)
				require.NoError(t, err)
				require.Equal(t, fileTypeBits, stat_t.Mode&syscall.S_IFMT)
				require.Equal(t, modeBits, stat_t.Mode&07777)
				if isDevice {
					require.Equal(t, dev, stat_t.Rdev)
				}
			})
		}
		t.Run("File type", func(t *testing.T) {
			testMknod(t, "socket", syscall.S_IFSOCK, 0644, 0)
			testMknod(t, "regular file", syscall.S_IFREG, 0644, 0)
			testMknod(t, "block device", syscall.S_IFBLK, 0644, unix.Mkdev(132, 4123))
			testMknod(t, "character device", syscall.S_IFCHR, 0644, unix.Mkdev(3421, 7623))
			testMknod(t, "FIFO", syscall.S_IFIFO, 0644, 0)
		})
		t.Run("Mode bits", func(t *testing.T) {
			for _, modeBits := range allModeBits {
				testMknod(t, fmt.Sprintf("mode bits %#.12o", modeBits), syscall.S_IFREG, modeBits, 0)
			}
		})
	})

	t.Run("WriteFile", func(t *testing.T) {
		for _, modeBits := range allModeBits {
			t.Run(fmt.Sprintf("Success mode=%#.12o", modeBits), func(t *testing.T) {
				name := filepath.Join(t.TempDir(), "foo")
				dataBytes := []byte("foo")
				data := bytes.NewReader(dataBytes)

				err := hst.WriteFile(ctx, name, data, modeBits)
				require.NoError(t, err)

				readDataBytes, err := os.ReadFile(name)
				if (modeBits & syscall.S_IRUSR) != 0 {
					require.NoError(t, err)
					require.Equal(t, dataBytes, readDataBytes)
				} else {
					require.ErrorIs(t, err, os.ErrPermission)
				}

				var stat_t syscall.Stat_t
				require.NoError(t, syscall.Lstat(name, &stat_t))
				require.Equal(t, modeBits, stat_t.Mode&07777)
			})
		}

		t.Run("path must be absolute", func(t *testing.T) {
			err := hst.WriteFile(ctx, "foo/bar", bytes.NewReader([]byte{}), 0600)
			require.ErrorContains(t, err, "path must be absolute")
		})
		t.Run("ErrPermission", func(t *testing.T) {
			skipIfRoot(t)
			err := hst.WriteFile(ctx, "/etc/foo", bytes.NewReader([]byte{}), 0600)
			require.ErrorIs(t, err, os.ErrPermission)
		})
		t.Run("ovewrite file", func(t *testing.T) {
			name := filepath.Join(t.TempDir(), "foo")
			var fileMode uint32 = 01607
			err := hst.WriteFile(ctx, name, bytes.NewReader([]byte{}), fileMode)
			require.NoError(t, err)
			dataBytes := []byte("foo")
			data := bytes.NewReader(dataBytes)
			var newFileMode uint32 = 02675
			err = hst.WriteFile(ctx, name, data, newFileMode)
			require.NoError(t, err)
			readDataBytes, err := os.ReadFile(name)
			require.NoError(t, err)
			require.Equal(t, dataBytes, readDataBytes)
			var stat_t syscall.Stat_t
			require.NoError(t, syscall.Lstat(name, &stat_t))
			require.Equal(t, newFileMode, stat_t.Mode&07777)
		})
		t.Run("is directory", func(t *testing.T) {
			name := filepath.Join(t.TempDir(), "foo")
			err := os.Mkdir(name, os.FileMode(0700))
			require.NoError(t, err)
			err = hst.WriteFile(ctx, name, bytes.NewReader([]byte{}), 0640)
			require.Error(t, err)
		})
		t.Run("ErrNotExist", func(t *testing.T) {
			name := filepath.Join(t.TempDir(), "foo", "bar")
			err := hst.WriteFile(ctx, name, bytes.NewReader([]byte{}), 0600)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})
}
