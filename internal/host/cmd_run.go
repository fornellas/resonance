package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alessio/shellescape"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

// This partially implements host.Host interface, with the exception of the following functions:
// Run, String and Close. Full implementtations of the host.Host interface can embed this struct,
// and just implement the remaining methods.
// The use case for this is for share code across host.Host implementations that solely rely
// on spawning commands via Run.
type cmdHost struct {
	Host host.Host
}

func (c cmdHost) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Chmod", "name", name, "mode", mode)

	cmd := host.Cmd{
		Path: "chmod",
		Args: []string{fmt.Sprintf("%o", mode), name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, c.Host, cmd)
	if err != nil {
		return err
	}
	if waitStatus.Success() {
		return nil
	}

	if strings.Contains(stderr, "Operation not permitted") {
		return os.ErrPermission
	}

	if strings.Contains(stderr, "No such file or directory") {
		return os.ErrNotExist
	}

	return fmt.Errorf(
		"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
		cmd, waitStatus.String(), stdout, stderr,
	)
}

func (c cmdHost) Chown(ctx context.Context, name string, uid, gid int) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Chown", "name", name, "uid", uid, "gid", gid)

	cmd := host.Cmd{
		Path: "chown",
		Args: []string{fmt.Sprintf("%d.%d", uid, gid), name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, c.Host, cmd)
	if err != nil {
		return err
	}
	if waitStatus.Success() {
		return nil
	}

	if strings.Contains(stderr, "Operation not permitted") {
		return os.ErrPermission
	}

	if strings.Contains(stderr, "No such file or directory") {
		return os.ErrNotExist
	}

	return fmt.Errorf(
		"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
		cmd, waitStatus.String(), stdout, stderr,
	)
}

func (c cmdHost) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lookup", "username", username)

	cmd := host.Cmd{
		Path: "cat",
		Args: []string{"/etc/passwd"},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, c.Host, cmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		return nil, fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	for i, line := range strings.Split(stdout, "\n") {
		if line == "" {
			continue
		}
		columns := strings.Split(line, ":")
		if len(columns) != 7 {
			return nil, fmt.Errorf("/etc/passwd:%d: unexpected number of columns %d: %#v", i, len(columns), line)
		}
		login := columns[0]
		if login != username {
			continue
		}
		// password := columns[1]
		uid := columns[2]
		gid := columns[3]
		name := columns[4]
		home := columns[5]
		// interpreter := columns[6]
		return &user.User{
			Uid:      uid,
			Gid:      gid,
			Username: login,
			Name:     name,
			HomeDir:  home,
		}, nil
	}
	return nil, user.UnknownUserError(username)
}

func (c cmdHost) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("LookupGroup", "name", name)

	cmd := host.Cmd{
		Path: "cat",
		Args: []string{"/etc/group"},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, c.Host, cmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		return nil, fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	for i, line := range strings.Split(stdout, "\n") {
		if line == "" {
			continue
		}
		columns := strings.Split(line, ":")
		if len(columns) != 4 {
			return nil, fmt.Errorf("/etc/group:%d: unexpected number of columns %d: %#v", i, len(columns), line)
		}
		group_name := columns[0]
		// password := columns[1]
		gid := columns[2]
		// user_list := columns[3]
		if name != group_name {
			continue
		}
		return &user.Group{
			Gid:  gid,
			Name: group_name,
		}, nil
	}
	return nil, user.UnknownGroupError(name)
}

func (c cmdHost) stat(ctx context.Context, name string) (string, error) {
	cmd := host.Cmd{
		Path: "stat",
		Args: []string{"--format=%d,%i,%h,%f,%u,%g,%t,%T,%s,%o,%b,%x,%y,%z", name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, c.Host, cmd)
	if err != nil {
		return "", err
	}
	if !waitStatus.Success() {
		if strings.Contains(stderr, "Permission denied") {
			return "", os.ErrPermission
		}
		if strings.Contains(stderr, "No such file or directory") {
			return "", os.ErrNotExist
		}
		return "", fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	return stdout, nil
}

func (c cmdHost) Lstat(ctx context.Context, name string) (host.FileInfo, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lstat", "name", name)

	stdout, err := c.stat(ctx, name)
	if err != nil {
		return host.FileInfo{}, err
	}

	tokens := strings.Split(strings.TrimRight(stdout, "\n"), ",")
	if len(tokens) != 14 {
		return host.FileInfo{}, fmt.Errorf("unable to parse stat output: %s", tokens)
	}

	// dev, err := strconv.ParseUint(tokens[0], 10, 64)
	// if err != nil {
	// 	return host.HostFileInfo{}, fmt.Errorf("unable to parse dev: %s", tokens[0])
	// }

	// ino, err := strconv.ParseUint(tokens[1], 10, 64)
	// if err != nil {
	// 	return host.HostFileInfo{}, fmt.Errorf("unable to parse ino: %s", tokens[1])
	// }

	// nlink, err := strconv.ParseUint(tokens[2], 10, 64)
	// if err != nil {
	// 	return host.HostFileInfo{}, fmt.Errorf("unable to parse nlink: %s", tokens[2])
	// }

	statMode, err := strconv.ParseUint(tokens[3], 16, 32)
	if err != nil {
		return host.FileInfo{}, fmt.Errorf("unable to parse mode: %s", tokens[3])
	}
	mode := fs.FileMode(uint32(statMode) & (uint32(fs.ModeType) | uint32(fs.ModePerm)))

	uid, err := strconv.ParseUint(tokens[4], 10, 32)
	if err != nil {
		return host.FileInfo{}, fmt.Errorf("unable to parse uid: %s", tokens[4])
	}

	gid, err := strconv.ParseUint(tokens[5], 10, 32)
	if err != nil {
		return host.FileInfo{}, fmt.Errorf("unable to parse gid: %s", tokens[5])
	}

	// fileInfo.stat_t.Rdev = column[7] // uint64

	size, err := strconv.ParseInt(tokens[8], 10, 64)
	if err != nil {
		return host.FileInfo{}, fmt.Errorf("unable to parse Size: %s", tokens[8])
	}

	// blksize, err := strconv.ParseInt(tokens[9], 10, 64)
	// if err != nil {
	// 	return host.HostFileInfo{}, fmt.Errorf("unable to parse blksize: %s", tokens[9])
	// }

	// fileInfo.stat_t.Blocks = column[10] // int64

	// atimTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700", tokens[11])
	// if err != nil {
	// 	return host.HostFileInfo{}, fmt.Errorf("unable to parse atim: %s: %w", tokens[11], err)
	// }

	modTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700", tokens[12])
	if err != nil {
		return host.FileInfo{}, fmt.Errorf("unable to parse modTime: %s: %w", tokens[12], err)
	}

	// ctimTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700", tokens[13])
	// if err != nil {
	// 	return host.HostFileInfo{}, fmt.Errorf("unable to parse ctim: %s: %w", tokens[13], err)
	// }

	return host.FileInfo{
		Name:     filepath.Base(name),
		Size:     size,
		FileMode: mode,
		ModTime:  modTime,
		Uid:      uint32(uid),
		Gid:      uint32(gid),
	}, nil
}

func (c cmdHost) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Mkdir", "name", name, "perm", perm)

	cmd := host.Cmd{
		Path: "mkdir",
		Args: []string{name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, c.Host, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		if strings.Contains(stderr, "Permission denied") {
			return os.ErrPermission
		}
		if strings.Contains(stderr, "File exists") {
			return os.ErrExist
		}
		if strings.Contains(stderr, "No such file or directory") {
			return os.ErrNotExist
		}
		return fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}

	return c.Chmod(ctx, name, perm)
}

func (c cmdHost) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("ReadFile", "name", name)

	cmd := host.Cmd{
		Path: "cat",
		Args: []string{name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, c.Host, cmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		if strings.Contains(stderr, "Permission denied") {
			return nil, os.ErrPermission
		}
		if strings.Contains(stderr, "No such file or directory") {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	return []byte(stdout), nil
}

func (c cmdHost) ReadDir(ctx context.Context, name string) ([]host.DirEntry, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("ReadDir", "name", name)

	cmd := host.Cmd{
		Path: "find",
		Args: []string{
			fmt.Sprintf("%s/", name),
			"-mindepth", "1",
			"-maxdepth", "1",
			"-printf", "%y %m %U %G %s %T+ %f\\0",
		},
		Env: []string{
			"TZ=GMT",
		},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, c.Host, cmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		if strings.Contains(stderr, "Permission denied") {
			return nil, os.ErrPermission
		}
		if strings.Contains(stderr, "No such file or directory") {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	entries := bytes.Split([]byte(stdout), []byte{0})
	var dirEntries []host.DirEntry

	for _, entry := range entries {
		fmt.Printf("enttry: %#v\n", string(entry))
		reader := strings.NewReader(string(entry))

		var dirEntry host.DirEntry
		var fileType rune
		var modYear int
		var modMonth int
		var modDay int
		var modHour int
		var modMinute int
		var modSecondF float64

		_, err := fmt.Fscanf(
			reader,
			"%c %o %d %d %d %d-%d-%d+%d:%d:%f %s\n",
			&fileType,
			&dirEntry.FileMode,
			&dirEntry.Uid,
			&dirEntry.Gid,
			&dirEntry.Size,
			&modYear,
			&modMonth,
			&modDay,
			&modHour,
			&modMinute,
			&modSecondF,
			&dirEntry.Name,
		)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error parsing output: %w: %#v", err, string(entry))
		}

		switch fileType {
		case 's':
			dirEntry.FileMode |= fs.ModeSocket
		case 'l':
			dirEntry.FileMode |= fs.ModeSymlink
		case 'r':
			continue
		case 'b':
			dirEntry.FileMode |= fs.ModeDevice
		case 'd':
			dirEntry.FileMode |= fs.ModeDir
		case 'c':
			dirEntry.FileMode |= fs.ModeCharDevice
		case 'f':
			dirEntry.FileMode |= fs.ModeNamedPipe
		}

		modSec, modNsec := math.Modf(modSecondF)
		modNsec = modNsec * 1000000000

		modTime := time.Date(modYear, time.Month(modMonth), modDay, modHour, modMinute, int(modSec), int(modNsec), time.UTC)

		dirEntry.ModTime = modTime.In(time.Local)

		dirEntries = append(dirEntries, dirEntry)
	}

	return dirEntries, nil
}

func (c cmdHost) rmdir(ctx context.Context, name string) error {
	cmd := host.Cmd{
		Path: "rmdir",
		Args: []string{name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, c.Host, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		if strings.Contains(stderr, "Permission denied") {
			return os.ErrPermission
		}
		return fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	return nil
}

func (c cmdHost) Remove(ctx context.Context, name string) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Remove", "name", name)

	cmd := host.Cmd{
		Path: "rm",
		Args: []string{name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, c.Host, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		if strings.Contains(stderr, "Is a directory") {
			return c.rmdir(ctx, name)
		}
		if strings.Contains(stderr, "Permission denied") {
			return os.ErrPermission
		}
		if strings.Contains(stderr, "No such file or directory") {
			return os.ErrNotExist
		}
		return fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	return nil
}

func (c cmdHost) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	logger := log.MustLogger(ctx)

	logger.Debug("WriteFile", "name", name, "data", data, "perm", perm)

	var chmod bool
	if _, err := c.Lstat(ctx, name); errors.Is(err, os.ErrNotExist) {
		chmod = true
	}
	cmd := host.Cmd{
		Path:  "sh",
		Args:  []string{"-c", fmt.Sprintf("cat > %s", shellescape.Quote(name))},
		Stdin: bytes.NewReader(data),
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, c.Host, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		if strings.Contains(stderr, "Is a directory") {
			return syscall.EISDIR
		}
		if strings.Contains(stderr, "Permission denied") {
			return os.ErrPermission
		}
		if strings.Contains(stderr, "Directory nonexistent") {
			return os.ErrNotExist
		}
		return fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	if chmod {
		return c.Chmod(ctx, name, perm)
	}
	return nil
}
