package host

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host/types"
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

//gocyclo:ignore
func testHost(
	t *testing.T,
	ctx context.Context,
	tempDirPrefix string,
	hst types.Host,
	hostString,
	hostType string,
) {
	testBaseHost(t, ctx, tempDirPrefix, hst, hostString, hostType)

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
		dir := tempDirWithPrefix(t, tempDirPrefix)
		name := filepath.Join(dir, "foo")
		file, err := os.Create(name)
		require.NoError(t, err)
		file.Close()
		t.Run("Success", func(t *testing.T) {
			var fileMode types.FileMode = 01257
			err = hst.Chmod(ctx, name, fileMode)
			require.NoError(t, err)
			var stat_t syscall.Stat_t
			require.NoError(t, syscall.Lstat(name, &stat_t))
			require.Equal(t, fileMode, types.FileMode(stat_t.Mode&07777))
		})
		t.Run("path must be absolute", func(t *testing.T) {
			err = hst.Chmod(ctx, "foo/bar", 0644)
			require.ErrorContains(t, err, "path must be absolute")
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.EPERM", func(t *testing.T) {
			skipIfRoot(t)
			err = hst.Chmod(ctx, "/tmp", 0)
			require.ErrorIs(t, err, syscall.EPERM)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.ENOENT", func(t *testing.T) {
			err = hst.Chmod(ctx, "/non-existent", 0)
			require.ErrorIs(t, err, syscall.ENOENT)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
	})

	t.Run("Lchown", func(t *testing.T) {
		prefix := t.TempDir()
		regularFilePath := filepath.Join(prefix, "regular")
		file, err := os.Create(regularFilePath)
		require.NoError(t, err)
		require.NoError(t, file.Close())
		fileInfo, err := os.Lstat(regularFilePath)
		require.NoError(t, err)
		regularFileStat_t := fileInfo.Sys().(*syscall.Stat_t)
		t.Run("Success", func(t *testing.T) {
			if isRoot(t) {
				symlinkPatht := filepath.Join(prefix, "symlink")
				require.NoError(t, os.Symlink(regularFilePath, symlinkPatht))

				var uid uint32 = 2341
				var gid uint32 = 2341
				require.NoError(t, hst.Lchown(ctx, symlinkPatht, uid, gid))

				fileInfo, err := os.Lstat(regularFilePath)
				require.NoError(t, err)
				symlinkStat_t := fileInfo.Sys().(*syscall.Stat_t)

				require.True(t, reflect.DeepEqual(symlinkStat_t, regularFileStat_t))

				fileInfo, err = os.Lstat(symlinkPatht)
				require.NoError(t, err)
				newSymlinkStat_t := fileInfo.Sys().(*syscall.Stat_t)
				require.Equal(t, uid, newSymlinkStat_t.Uid)
				require.Equal(t, gid, newSymlinkStat_t.Gid)
			} else {
				err = hst.Lchown(ctx, regularFilePath, regularFileStat_t.Uid, regularFileStat_t.Gid)
				require.NoError(t, err)
			}
		})
		t.Run("path must be absolute", func(t *testing.T) {
			err = hst.Lchown(ctx, "foo/bar", 0, 0)
			require.ErrorContains(t, err, "path must be absolute")
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.EPERM", func(t *testing.T) {
			skipIfRoot(t)
			err = hst.Lchown(ctx, regularFilePath, 0, 0)
			require.ErrorIs(t, err, syscall.EPERM)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.ENOENT", func(t *testing.T) {
			err = hst.Lchown(ctx, "/non-existent", 0, 0)
			require.ErrorIs(t, err, syscall.ENOENT)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
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
			dir := tempDirWithPrefix(t, tempDirPrefix)
			name := filepath.Join(dir, "foo")
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
						dir := tempDirWithPrefix(t, tempDirPrefix)
						socketPath := filepath.Join(dir, "socket")
						err = syscall.Mknod(socketPath, syscall.S_IFSOCK, 0)
						require.NoError(t, err)
						socketStat_t, err := hst.Lstat(ctx, socketPath)
						require.NoError(t, err)
						require.True(t, socketStat_t.Mode&syscall.S_IFMT == syscall.S_IFSOCK)
					})
					t.Run("symbolic link", func(t *testing.T) {
						dir := tempDirWithPrefix(t, tempDirPrefix)
						linkPath := filepath.Join(dir, "symlink")
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
						dir := tempDirWithPrefix(t, tempDirPrefix)
						fifoPath := filepath.Join(dir, "fifo")
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
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.EACCES", func(t *testing.T) {
			skipIfRoot(t)
			_, err := hst.Lstat(ctx, "/etc/ssl/private/foo")
			require.ErrorIs(t, err, syscall.EACCES)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.ENOENT", func(t *testing.T) {
			_, err := hst.Lstat(ctx, "/non-existent")
			require.ErrorIs(t, err, syscall.ENOENT)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
	})

	t.Run("ReadDir", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {

			dir := tempDirWithPrefix(t, tempDirPrefix)

			expectedTypeMap := map[string]uint8{}

			// socket
			socketPath := filepath.Join(dir, "socket")
			err := syscall.Mknod(socketPath, syscall.S_IFSOCK, 0)
			require.NoError(t, err)
			expectedTypeMap["socket"] = syscall.DT_SOCK

			// symbolic link
			linkPath := filepath.Join(dir, "symlink")
			require.NoError(t, syscall.Symlink("foo", linkPath))
			expectedTypeMap["symlink"] = syscall.DT_LNK

			// regular file
			file, err := os.Create(filepath.Join(dir, "regular"))
			require.NoError(t, err)
			defer file.Close()
			expectedTypeMap["regular"] = syscall.DT_REG

			// block device: can't test without root

			// directory
			require.NoError(t, os.Mkdir(filepath.Join(dir, "directory"), os.FileMode(0700)))
			expectedTypeMap["directory"] = syscall.DT_DIR

			// character device: can't test without root

			// FIFO
			require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "FIFO"), 0644))
			expectedTypeMap["FIFO"] = syscall.DT_FIFO

			dirEntResultCh, cancel := hst.ReadDir(ctx, dir)
			defer cancel()

			inodeMap := map[uint64]bool{}
			for dirEntResult := range dirEntResultCh {
				require.NoError(t, dirEntResult.Error)
				dirEnt := dirEntResult.DirEnt
				require.NotContains(t, inodeMap, dirEnt.Ino)
				inodeMap[dirEnt.Ino] = true
				require.Contains(t, expectedTypeMap, dirEnt.Name)
				require.Equal(t, expectedTypeMap[dirEnt.Name], dirEnt.Type)
				delete(expectedTypeMap, dirEnt.Name)
			}
			require.Empty(t, expectedTypeMap)
		})
		t.Run("path must be absolute", func(t *testing.T) {
			dirEntResultCh, cancel := hst.ReadDir(ctx, "foo")
			defer cancel()
			direntResult := <-dirEntResultCh
			require.ErrorContains(t, direntResult.Error, "path must be absolute")
			var pathError *fs.PathError
			require.ErrorAs(t, direntResult.Error, &pathError)
		})
		t.Run("syscall.EACCES", func(t *testing.T) {
			skipIfRoot(t)
			dirEntResultCh, cancel := hst.ReadDir(ctx, "/etc/ssl/private/foo")
			defer cancel()
			direntResult := <-dirEntResultCh
			require.ErrorIs(t, direntResult.Error, syscall.EACCES)
			var pathError *fs.PathError
			require.ErrorAs(t, direntResult.Error, &pathError)
		})
		t.Run("syscall.ENOENT", func(t *testing.T) {
			dirEntResultCh, cancel := hst.ReadDir(ctx, "/non-existent")
			defer cancel()
			direntResult := <-dirEntResultCh
			require.ErrorIs(t, direntResult.Error, syscall.ENOENT)
			var pathError *fs.PathError
			require.ErrorAs(t, direntResult.Error, &pathError)
		})
	})

	t.Run("Mkdir", func(t *testing.T) {
		for _, fileMode := range allModeBits {
			t.Run(fmt.Sprintf("Success mode=%#.12o", fileMode), func(t *testing.T) {
				dir := tempDirWithPrefix(t, tempDirPrefix)
				name := filepath.Join(dir, "foo")
				err := hst.Mkdir(ctx, name, types.FileMode(fileMode))
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
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.EACCES", func(t *testing.T) {
			skipIfRoot(t)
			err := hst.Mkdir(ctx, "/etc/foo", 0750)
			require.ErrorIs(t, err, syscall.EACCES)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.EEXIST", func(t *testing.T) {
			dir := tempDirWithPrefix(t, tempDirPrefix)
			name := filepath.Join(dir, "foo")
			err := hst.Mkdir(ctx, name, 0750)
			require.NoError(t, err)
			err = hst.Mkdir(ctx, name, 0750)
			require.ErrorIs(t, err, syscall.EEXIST)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.ENOENT", func(t *testing.T) {
			dir := tempDirWithPrefix(t, tempDirPrefix)
			name := filepath.Join(dir, "foo", "bar")
			err := hst.Mkdir(ctx, name, 0750)
			require.ErrorIs(t, err, syscall.ENOENT)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
	})

	t.Run("ReadFile", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			t.Run("with contents", func(t *testing.T) {
				dir := tempDirWithPrefix(t, tempDirPrefix)
				name := filepath.Join(dir, "foo")
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
			t.Run("empty", func(t *testing.T) {
				dir := tempDirWithPrefix(t, tempDirPrefix)
				name := filepath.Join(dir, "foo")
				data := []byte{}
				err := os.WriteFile(name, data, os.FileMode(0600))
				require.NoError(t, err)
				fileReadCloser, err := hst.ReadFile(ctx, name)
				require.NoError(t, err)
				readData, err := io.ReadAll(fileReadCloser)
				assert.NoError(t, err)
				assert.Equal(t, data, readData)
				require.NoError(t, fileReadCloser.Close())
			})
		})
		t.Run("path must be absolute", func(t *testing.T) {
			fileReadCloser, err := hst.ReadFile(ctx, "foo/bar")
			defer func() {
				if err == nil {
					require.NoError(t, fileReadCloser.Close())
				}
			}()
			require.ErrorContains(t, err, "path must be absolute")
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.EACCES", func(t *testing.T) {
			fileReadCloser, err := hst.ReadFile(ctx, "/etc/shadow")
			defer func() {
				if err == nil {
					require.NoError(t, fileReadCloser.Close())
				}
			}()
			skipIfRoot(t)
			require.ErrorIs(t, err, syscall.EACCES)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.ENOENT", func(t *testing.T) {
			fileReadCloser, err := hst.ReadFile(ctx, "/non-existent")
			defer func() {
				if err == nil {
					require.NoError(t, fileReadCloser.Close())
				}
			}()
			require.ErrorIs(t, err, syscall.ENOENT)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
	})

	t.Run("Symlink", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			dir := tempDirWithPrefix(t, tempDirPrefix)
			newname := filepath.Join(dir, "newname")
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
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.EACCES", func(t *testing.T) {
			skipIfRoot(t)
			newname := "/etc/foo"
			oldname := "foo/bar"
			err := hst.Symlink(ctx, oldname, newname)
			require.ErrorIs(t, err, syscall.EACCES)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.EEXIST", func(t *testing.T) {
			dir := tempDirWithPrefix(t, tempDirPrefix)
			newname := filepath.Join(dir, "newname")
			oldname := "foo/bar"
			err := hst.Symlink(ctx, oldname, newname)
			require.NoError(t, err)
			err = hst.Symlink(ctx, oldname, newname)
			require.ErrorIs(t, err, syscall.EEXIST)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.ENOENT", func(t *testing.T) {
			oldname := "foo/bar"
			newname := "/bad/path"
			err := hst.Symlink(ctx, oldname, newname)
			require.ErrorIs(t, err, syscall.ENOENT)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
	})

	t.Run("Readlink", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			dir := tempDirWithPrefix(t, tempDirPrefix)
			target := filepath.Join(dir, "target")
			err := os.WriteFile(target, []byte("content"), 0644)
			require.NoError(t, err)
			symlink := filepath.Join(dir, "symlink")
			err = os.Symlink(target, symlink)
			require.NoError(t, err)
			result, err := hst.Readlink(ctx, symlink)
			require.NoError(t, err)
			require.Equal(t, target, result)
		})
		t.Run("path must be absolute", func(t *testing.T) {
			_, err := hst.Readlink(ctx, "foo/bar")
			require.ErrorContains(t, err, "path must be absolute")
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.ENOENT", func(t *testing.T) {
			_, err := hst.Readlink(ctx, "/non-existent")
			require.ErrorIs(t, err, syscall.ENOENT)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
	})

	t.Run("Remove", func(t *testing.T) {
		t.Run("Success file", func(t *testing.T) {
			dir := tempDirWithPrefix(t, tempDirPrefix)
			name := filepath.Join(dir, "foo")
			file, err := os.Create(name)
			require.NoError(t, err)
			file.Close()
			err = hst.Remove(ctx, name)
			require.NoError(t, err)
			_, err = os.Lstat(name)
			require.ErrorIs(t, err, syscall.ENOENT)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("Success dir", func(t *testing.T) {
			dir := tempDirWithPrefix(t, tempDirPrefix)
			name := filepath.Join(dir, "foo")
			err := os.Mkdir(name, 0700)
			require.NoError(t, err)
			err = hst.Remove(ctx, name)
			require.NoError(t, err)
			_, err = os.Lstat(name)
			require.ErrorIs(t, err, syscall.ENOENT)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("path must be absolute", func(t *testing.T) {
			err := hst.Remove(ctx, "foo/bar")
			require.ErrorContains(t, err, "path must be absolute")
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.EACCES file", func(t *testing.T) {
			skipIfRoot(t)
			err := hst.Remove(ctx, "/bin/ls")
			require.ErrorIs(t, err, syscall.EACCES)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.EACCES dir", func(t *testing.T) {
			skipIfRoot(t)
			err := hst.Remove(ctx, "/bin")
			require.ErrorIs(t, err, syscall.EACCES)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.ENOENT", func(t *testing.T) {
			err := hst.Remove(ctx, "/non-existent")
			require.ErrorIs(t, err, syscall.ENOENT)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
	})

	t.Run("Mknod", func(t *testing.T) {
		testMknod := func(t *testing.T, name string, fileType, modeBits uint32, dev types.FileDevice) {
			t.Run(name, func(t *testing.T) {
				dir := tempDirWithPrefix(t, tempDirPrefix)
				path := filepath.Join(dir, name)
				var fileTypeBits uint32 = fileType
				var mode types.FileMode = types.FileMode(fileTypeBits | modeBits)

				var isDevice bool
				switch fileType & syscall.S_IFMT {
				case syscall.S_IFBLK, syscall.S_IFCHR:
					isDevice = true
				}

				err := hst.Mknod(ctx, path, mode, dev)
				if isDevice && !isRoot(t) {
					require.ErrorIs(t, err, syscall.EPERM)
					var pathError *fs.PathError
					require.ErrorAs(t, err, &pathError)
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
			testMknod(t, "block device", syscall.S_IFBLK, 0644, types.FileDevice(unix.Mkdev(132, 4123)))
			testMknod(t, "character device", syscall.S_IFCHR, 0644, types.FileDevice(unix.Mkdev(3421, 7623)))
			testMknod(t, "FIFO", syscall.S_IFIFO, 0644, 0)
		})
		t.Run("Mode bits", func(t *testing.T) {
			for _, modeBits := range allModeBits {
				testMknod(t, fmt.Sprintf("mode bits %#.12o", modeBits), syscall.S_IFREG, modeBits, 0)
			}
		})
	})

	testFileCommon := func(
		t *testing.T,
		fileFn func(ctx context.Context, name string, data io.Reader, mode types.FileMode) error,
	) {
		for _, modeBits := range allModeBits {
			t.Run(fmt.Sprintf("Success mode=%#.12o", modeBits), func(t *testing.T) {
				dir := tempDirWithPrefix(t, tempDirPrefix)
				name := filepath.Join(dir, "foo")
				dataBytes := []byte("foo")
				data := bytes.NewReader(dataBytes)

				err := fileFn(ctx, name, data, types.FileMode(modeBits))
				require.NoError(t, err)

				readDataBytes, err := os.ReadFile(name)
				if (modeBits&syscall.S_IRUSR) != 0 || isRoot(t) {
					require.NoError(t, err)
					require.Equal(t, dataBytes, readDataBytes)
				} else {
					require.ErrorIs(t, err, syscall.EACCES)
				}

				var stat_t syscall.Stat_t
				require.NoError(t, syscall.Lstat(name, &stat_t))
				require.Equal(t, modeBits, stat_t.Mode&07777)
			})
		}

		t.Run("path must be absolute", func(t *testing.T) {
			err := fileFn(ctx, "foo/bar", bytes.NewReader([]byte{}), 0600)
			require.ErrorContains(t, err, "path must be absolute")
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.EACCES", func(t *testing.T) {
			skipIfRoot(t)
			err := fileFn(ctx, "/etc/foo", bytes.NewReader([]byte{}), 0600)
			require.ErrorIs(t, err, syscall.EACCES)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("is directory", func(t *testing.T) {
			dir := tempDirWithPrefix(t, tempDirPrefix)
			name := filepath.Join(dir, "foo")
			err := os.Mkdir(name, os.FileMode(0700))
			require.NoError(t, err)
			err = fileFn(ctx, name, bytes.NewReader([]byte{}), 0640)
			require.Error(t, err)
			require.ErrorIs(t, err, syscall.EISDIR)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
		t.Run("syscall.ENOENT", func(t *testing.T) {
			dir := tempDirWithPrefix(t, tempDirPrefix)
			name := filepath.Join(dir, "foo", "bar")
			err := fileFn(ctx, name, bytes.NewReader([]byte{}), 0600)
			require.ErrorIs(t, err, syscall.ENOENT)
			var pathError *fs.PathError
			require.ErrorAs(t, err, &pathError)
		})
	}

	t.Run("WriteFile", func(t *testing.T) {
		testFileCommon(t, hst.WriteFile)

		t.Run("ovewrite file", func(t *testing.T) {
			dir := tempDirWithPrefix(t, tempDirPrefix)
			name := filepath.Join(dir, "foo")
			oldDataBytes := []byte("old")
			oldData := bytes.NewReader(oldDataBytes)
			var fileMode types.FileMode = 01607
			err := hst.WriteFile(ctx, name, oldData, fileMode)
			require.NoError(t, err)
			newDataBytes := []byte("new")
			newData := bytes.NewReader(newDataBytes)
			var newFileMode types.FileMode = 02675
			err = hst.WriteFile(ctx, name, newData, newFileMode)
			require.NoError(t, err)
			readDataBytes, err := os.ReadFile(name)
			require.NoError(t, err)
			require.Equal(t, newDataBytes, readDataBytes)
			var stat_t syscall.Stat_t
			require.NoError(t, syscall.Lstat(name, &stat_t))
			require.Equal(t, newFileMode, types.FileMode(stat_t.Mode&07777))
		})
	})

	t.Run("AppendFile", func(t *testing.T) {
		testFileCommon(t, hst.AppendFile)

		t.Run("append file", func(t *testing.T) {
			dir := tempDirWithPrefix(t, tempDirPrefix)
			name := filepath.Join(dir, "foo")
			oldDataBytes := []byte("old")
			oldData := bytes.NewReader(oldDataBytes)
			var fileMode types.FileMode = 01607
			err := hst.AppendFile(ctx, name, oldData, fileMode)
			require.NoError(t, err)
			newDataBytes := []byte("new")
			newData := bytes.NewReader(newDataBytes)
			var newFileMode types.FileMode = 02675
			err = hst.AppendFile(ctx, name, newData, newFileMode)
			require.NoError(t, err)
			readDataBytes, err := os.ReadFile(name)
			require.NoError(t, err)
			require.Equal(t, append(oldDataBytes, newDataBytes...), readDataBytes)
			var stat_t syscall.Stat_t
			require.NoError(t, syscall.Lstat(name, &stat_t))
			require.Equal(t, newFileMode, types.FileMode(stat_t.Mode&07777))
		})
	})
}
