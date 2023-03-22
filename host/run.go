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
)

// Run implements Host interface by having all methods rely on an underlying Host.Run.
type Run struct {
	Host Host
}

func (r Run) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	cmd := Cmd{
		Path:  "chmod",
		Args:  []string{fmt.Sprintf("%o", mode), name},
		Stdin: os.Stdin,
	}
	waitStatus, stdout, stderr, err := r.Host.Run(ctx, cmd)
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
		"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
		cmd, waitStatus, stdout, stderr,
	)
}

func (r Run) Chown(ctx context.Context, name string, uid, gid int) error {
	cmd := Cmd{
		Path:  "chown",
		Args:  []string{fmt.Sprintf("%d.%d", uid, gid), name},
		Stdin: os.Stdin,
	}
	waitStatus, stdout, stderr, err := r.Host.Run(ctx, cmd)
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
		"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
		cmd, waitStatus, stdout, stderr,
	)
}

func (r Run) Lookup(ctx context.Context, username string) (*user.User, error) {
	cmd := Cmd{
		Path:  "cat",
		Args:  []string{"/etc/passwd"},
		Stdin: os.Stdin,
	}
	waitStatus, stdout, stderr, err := r.Host.Run(ctx, cmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		return nil, fmt.Errorf(
			"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus, stdout, stderr,
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
		// password := columns[1]
		uid := columns[2]
		gid := columns[3]
		name := columns[4]
		home := columns[5]
		// interpreter := columns[6]
		if name != username {
			continue
		}
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

func (r Run) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	cmd := Cmd{
		Path:  "cat",
		Args:  []string{"/etc/group"},
		Stdin: os.Stdin,
	}
	waitStatus, stdout, stderr, err := r.Host.Run(ctx, cmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		return nil, fmt.Errorf(
			"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus, stdout, stderr,
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

type FileInfo struct {
	name   string
	stat_t syscall.Stat_t
}

func (fi FileInfo) Name() string {
	return fi.name
}
func (fi FileInfo) Size() int64 {
	return fi.stat_t.Size
}

func (fi FileInfo) Mode() fs.FileMode {
	return fs.FileMode(fi.stat_t.Mode & (uint32(fs.ModeType) | uint32(fs.ModePerm)))
}

func (fi FileInfo) ModTime() time.Time {
	return time.Unix(fi.stat_t.Mtim.Sec, fi.stat_t.Mtim.Nsec)
}
func (fi FileInfo) IsDir() bool {
	return (fi.Mode() & fs.ModeDir) > 0
}
func (fi FileInfo) Sys() any {
	return &fi.stat_t
}

func (r Run) stat(ctx context.Context, name string) (string, error) {
	cmd := Cmd{
		Path:  "stat",
		Args:  []string{"--format=%d,%i,%h,%f,%u,%g,%t,%T,%s,%o,%b,%x,%y,%z", name},
		Stdin: os.Stdin,
	}
	waitStatus, stdout, stderr, err := r.Host.Run(ctx, cmd)
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
			"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus, stdout, stderr,
		)
	}
	return stdout, nil
}

func (r Run) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	stdout, err := r.stat(ctx, name)
	if err != nil {
		return nil, err
	}

	tokens := strings.Split(strings.TrimRight(stdout, "\n"), ",")
	if len(tokens) != 14 {
		return nil, fmt.Errorf("unable to parse stat output: %s", tokens)
	}

	fileInfo := &FileInfo{name: filepath.Base(name)}

	dev, err := strconv.ParseUint(tokens[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse dev: %s", tokens[0])
	}
	fileInfo.stat_t.Dev = dev

	ino, err := strconv.ParseUint(tokens[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ino: %s", tokens[1])
	}
	fileInfo.stat_t.Ino = ino

	nlink, err := strconv.ParseUint(tokens[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse nlink: %s", tokens[2])
	}
	fileInfo.stat_t.Nlink = nlink

	mode, err := strconv.ParseUint(tokens[3], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("unable to parse mode: %s", tokens[3])
	}
	fileInfo.stat_t.Mode = uint32(mode)

	uid, err := strconv.ParseUint(tokens[4], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("unable to parse uid: %s", tokens[4])
	}
	fileInfo.stat_t.Uid = uint32(uid)

	gid, err := strconv.ParseUint(tokens[5], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("unable to parse gid: %s", tokens[5])
	}
	fileInfo.stat_t.Gid = uint32(gid)

	// fileInfo.stat_t.X__pad0 = column[6] // int32

	// fileInfo.stat_t.Rdev = column[7] // uint64

	// fileInfo.stat_t.Size = column[8] // int64

	blksize, err := strconv.ParseInt(tokens[9], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse blksize: %s", tokens[9])
	}
	fileInfo.stat_t.Blksize = blksize

	// fileInfo.stat_t.Blocks = column[10] // int64

	atimTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700", tokens[11])
	if err != nil {
		return nil, fmt.Errorf("unable to parse atim: %s: %w", tokens[11], err)
	}
	fileInfo.stat_t.Atim = syscall.Timespec{
		Sec:  atimTime.Unix(),
		Nsec: atimTime.UnixNano() % 1000000000,
	}

	mtimTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700", tokens[12])
	if err != nil {
		return nil, fmt.Errorf("unable to parse mtim: %s: %w", tokens[12], err)
	}
	fileInfo.stat_t.Mtim = syscall.Timespec{
		Sec:  mtimTime.Unix(),
		Nsec: mtimTime.UnixNano() % 1000000000,
	}

	ctimTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700", tokens[13])
	if err != nil {
		return nil, fmt.Errorf("unable to parse ctim: %s: %w", tokens[13], err)
	}
	fileInfo.stat_t.Ctim = syscall.Timespec{
		Sec:  ctimTime.Unix(),
		Nsec: ctimTime.UnixNano() % 1000000000,
	}

	// fileInfo.stat_t.X__unused = column[14] // [3]int64

	return fileInfo, nil
}

func (r Run) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	cmd := Cmd{
		Path:  "mkdir",
		Args:  []string{name},
		Stdin: os.Stdin,
	}
	waitStatus, stdout, stderr, err := r.Host.Run(ctx, cmd)
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
			"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus, stdout, stderr,
		)
	}

	return r.Chmod(ctx, name, perm)
}

func (r Run) ReadFile(ctx context.Context, name string) ([]byte, error) {
	cmd := Cmd{
		Path:  "cat",
		Args:  []string{name},
		Stdin: os.Stdin,
	}
	waitStatus, stdout, stderr, err := r.Host.Run(ctx, cmd)
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
			"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus, stdout, stderr,
		)
	}
	return []byte(stdout), nil
}

func (r Run) rmdir(ctx context.Context, name string) error {
	cmd := Cmd{
		Path:  "rmdir",
		Args:  []string{name},
		Stdin: os.Stdin,
	}
	waitStatus, stdout, stderr, err := r.Host.Run(ctx, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		if strings.Contains(stderr, "Permission denied") {
			return os.ErrPermission
		}
		return fmt.Errorf(
			"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus, stdout, stderr,
		)
	}
	return nil
}

func (r Run) Remove(ctx context.Context, name string) error {
	cmd := Cmd{
		Path:  "rm",
		Args:  []string{name},
		Stdin: os.Stdin,
	}
	waitStatus, stdout, stderr, err := r.Host.Run(ctx, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		if strings.Contains(stderr, "Is a directory") {
			return r.rmdir(ctx, name)
		}
		if strings.Contains(stderr, "Permission denied") {
			return os.ErrPermission
		}
		if strings.Contains(stderr, "No such file or directory") {
			return os.ErrNotExist
		}
		return fmt.Errorf(
			"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus, stdout, stderr,
		)
	}
	return nil
}

func (r Run) Run(ctx context.Context, cmd Cmd) (WaitStatus, string, string, error) {
	return r.Host.Run(ctx, cmd)
}

func (r Run) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	var chmod bool
	if _, err := r.Lstat(ctx, name); errors.Is(err, os.ErrNotExist) {
		chmod = true
	}
	cmd := Cmd{
		Path:  "sh",
		Args:  []string{"-c", fmt.Sprintf("cat > %s", shellescape.Quote(name))},
		Stdin: bytes.NewReader(data),
	}
	waitStatus, stdout, stderr, err := r.Host.Run(ctx, cmd)
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
			"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus, stdout, stderr,
		)
	}
	if chmod {
		return r.Chmod(ctx, name, perm)
	}
	return nil
}

func (r Run) String() string {
	return r.Host.String()
}
