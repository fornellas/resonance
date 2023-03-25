package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alessio/shellescape"

	"github.com/fornellas/resonance/log"
)

type baseRun struct {
	Host Host
}

func (br baseRun) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Chmod %v %s", mode, name)
	nestedCtx := log.IndentLogger(ctx)

	cmd := Cmd{
		Path: "chmod",
		Args: []string{fmt.Sprintf("%o", mode), name},
	}
	waitStatus, stdout, stderr, err := Run(nestedCtx, br.Host, cmd)
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

func (br baseRun) Chown(ctx context.Context, name string, uid, gid int) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Chown %v %v %s", uid, gid, name)
	nestedCtx := log.IndentLogger(ctx)

	cmd := Cmd{
		Path: "chown",
		Args: []string{fmt.Sprintf("%d.%d", uid, gid), name},
	}
	waitStatus, stdout, stderr, err := Run(nestedCtx, br.Host, cmd)
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

func (br baseRun) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Lookup %s", username)
	nestedCtx := log.IndentLogger(ctx)

	cmd := Cmd{
		Path: "cat",
		Args: []string{"/etc/passwd"},
	}
	waitStatus, stdout, stderr, err := Run(nestedCtx, br.Host, cmd)
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

func (br baseRun) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("LookupGroup %s", name)
	nestedCtx := log.IndentLogger(ctx)

	cmd := Cmd{
		Path: "cat",
		Args: []string{"/etc/group"},
	}
	waitStatus, stdout, stderr, err := Run(nestedCtx, br.Host, cmd)
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

func (br baseRun) stat(ctx context.Context, name string) (string, error) {
	cmd := Cmd{
		Path: "stat",
		Args: []string{"--format=%d,%i,%h,%f,%u,%g,%t,%T,%s,%o,%b,%x,%y,%z", name},
	}
	waitStatus, stdout, stderr, err := Run(ctx, br.Host, cmd)
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

func (br baseRun) Lstat(ctx context.Context, name string) (HostFileInfo, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Lstat %s", name)
	nestedCtx := log.IndentLogger(ctx)

	stdout, err := br.stat(nestedCtx, name)
	if err != nil {
		return HostFileInfo{}, err
	}

	tokens := strings.Split(strings.TrimRight(stdout, "\n"), ",")
	if len(tokens) != 14 {
		return HostFileInfo{}, fmt.Errorf("unable to parse stat output: %s", tokens)
	}

	// dev, err := strconv.ParseUint(tokens[0], 10, 64)
	// if err != nil {
	// 	return HostFileInfo{}, fmt.Errorf("unable to parse dev: %s", tokens[0])
	// }

	// ino, err := strconv.ParseUint(tokens[1], 10, 64)
	// if err != nil {
	// 	return HostFileInfo{}, fmt.Errorf("unable to parse ino: %s", tokens[1])
	// }

	// nlink, err := strconv.ParseUint(tokens[2], 10, 64)
	// if err != nil {
	// 	return HostFileInfo{}, fmt.Errorf("unable to parse nlink: %s", tokens[2])
	// }

	statMode, err := strconv.ParseUint(tokens[3], 16, 32)
	if err != nil {
		return HostFileInfo{}, fmt.Errorf("unable to parse mode: %s", tokens[3])
	}
	mode := fs.FileMode(uint32(statMode) & (uint32(fs.ModeType) | uint32(fs.ModePerm)))

	uid, err := strconv.ParseUint(tokens[4], 10, 32)
	if err != nil {
		return HostFileInfo{}, fmt.Errorf("unable to parse uid: %s", tokens[4])
	}

	gid, err := strconv.ParseUint(tokens[5], 10, 32)
	if err != nil {
		return HostFileInfo{}, fmt.Errorf("unable to parse gid: %s", tokens[5])
	}

	// fileInfo.stat_t.Rdev = column[7] // uint64

	size, err := strconv.ParseInt(tokens[8], 10, 64)
	if err != nil {
		return HostFileInfo{}, fmt.Errorf("unable to parse Size: %s", tokens[8])
	}

	// blksize, err := strconv.ParseInt(tokens[9], 10, 64)
	// if err != nil {
	// 	return HostFileInfo{}, fmt.Errorf("unable to parse blksize: %s", tokens[9])
	// }

	// fileInfo.stat_t.Blocks = column[10] // int64

	// atimTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700", tokens[11])
	// if err != nil {
	// 	return HostFileInfo{}, fmt.Errorf("unable to parse atim: %s: %w", tokens[11], err)
	// }

	modTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700", tokens[12])
	if err != nil {
		return HostFileInfo{}, fmt.Errorf("unable to parse modTime: %s: %w", tokens[12], err)
	}

	isDir := (uint32(statMode) & uint32(fs.ModeDir)) > 0

	// ctimTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700", tokens[13])
	// if err != nil {
	// 	return HostFileInfo{}, fmt.Errorf("unable to parse ctim: %s: %w", tokens[13], err)
	// }

	return NewHostFileInfo(
		filepath.Base(name),
		size,
		mode,
		modTime,
		isDir,
		uint32(uid),
		uint32(gid),
	), nil
}

func (br baseRun) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Mkdir %s", name)
	nestedCtx := log.IndentLogger(ctx)

	cmd := Cmd{
		Path: "mkdir",
		Args: []string{name},
	}
	waitStatus, stdout, stderr, err := Run(nestedCtx, br.Host, cmd)
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

	return br.Chmod(ctx, name, perm)
}

func (br baseRun) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("ReadFile %s", name)
	nestedCtx := log.IndentLogger(ctx)

	cmd := Cmd{
		Path: "cat",
		Args: []string{name},
	}
	waitStatus, stdout, stderr, err := Run(nestedCtx, br.Host, cmd)
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

func (br baseRun) rmdir(ctx context.Context, name string) error {
	cmd := Cmd{
		Path: "rmdir",
		Args: []string{name},
	}
	waitStatus, stdout, stderr, err := Run(ctx, br.Host, cmd)
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

func (br baseRun) Remove(ctx context.Context, name string) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Remove %s", name)
	nestedCtx := log.IndentLogger(ctx)

	cmd := Cmd{
		Path: "rm",
		Args: []string{name},
	}
	waitStatus, stdout, stderr, err := Run(nestedCtx, br.Host, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		if strings.Contains(stderr, "Is a directory") {
			return br.rmdir(nestedCtx, name)
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

func (br baseRun) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("WriteFile %s %v", name, perm)
	nestedCtx := log.IndentLogger(ctx)

	var chmod bool
	if _, err := br.Lstat(nestedCtx, name); errors.Is(err, os.ErrNotExist) {
		chmod = true
	}
	cmd := Cmd{
		Path:  "sh",
		Args:  []string{"-c", fmt.Sprintf("cat > %s", shellescape.Quote(name))},
		Stdin: bytes.NewReader(data),
	}
	waitStatus, stdout, stderr, err := Run(nestedCtx, br.Host, cmd)
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
		return br.Chmod(nestedCtx, name, perm)
	}
	return nil
}
