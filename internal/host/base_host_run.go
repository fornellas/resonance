package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"al.essio.dev/pkg/shellescape"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

type statField struct {
	format string
	fn     func(string, *host.Stat_t) error
}

func NewStatField(format string, fn func(value string, stat_t *host.Stat_t) error) statField {
	return statField{format: format, fn: fn}
}

var statTimeFormat = "2006-01-02 15:04:05.999999999 -0700"

var statFields = []statField{
	// device number in decimal (st_dev)
	NewStatField("%d", func(value string, stat_t *host.Stat_t) error {
		var err error
		stat_t.Dev, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse dev: %s", value)
		}
		return nil
	}),
	// inode number
	NewStatField("%i", func(value string, stat_t *host.Stat_t) error {
		var err error
		stat_t.Ino, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse ino: %s", value)
		}
		return nil
	}),
	// number of hard links
	NewStatField("%h", func(value string, stat_t *host.Stat_t) error {
		var err error
		stat_t.Nlink, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse nlink: %s", value)
		}
		return nil
	}),
	// raw mode in hex
	NewStatField("%f", func(value string, stat_t *host.Stat_t) error {
		mode64, err := strconv.ParseUint(value, 16, 32)
		if err != nil {
			return fmt.Errorf("unable to parse mode: %s", value)
		}
		stat_t.Mode = uint32(mode64)
		return nil
	}),
	// user ID of owner
	NewStatField("%u", func(value string, stat_t *host.Stat_t) error {
		uid64, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("unable to parse uid: %s", value)
		}
		stat_t.Uid = uint32(uid64)
		return nil
	}),
	// group ID of owner
	NewStatField("%g", func(value string, stat_t *host.Stat_t) error {
		gid64, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("unable to parse gid: %s", value)
		}
		stat_t.Gid = uint32(gid64)
		return nil
	}),
	// device type in decimal (st_rdev)
	NewStatField("%r", func(value string, stat_t *host.Stat_t) error {
		if value == "?" {
			return nil
		}
		var err error
		stat_t.Rdev, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse rdev: %s", value)
		}
		return nil
	}),
	// total size, in bytes
	NewStatField("%s", func(value string, stat_t *host.Stat_t) error {
		var err error
		stat_t.Size, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse size: %s", value)
		}
		return nil
	}),
	// the size in bytes of each block reported by %b
	NewStatField("%B", func(value string, stat_t *host.Stat_t) error {
		var err error
		stat_t.Blksize, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse blksize: %s", value)
		}
		return nil
	}),
	// number of blocks allocated (see %B)
	NewStatField("%b", func(value string, stat_t *host.Stat_t) error {
		var err error
		stat_t.Blocks, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse blocks: %s", value)
		}
		return nil
	}),
	// time of last access, human-readable
	NewStatField("%x", func(value string, stat_t *host.Stat_t) error {
		var err error
		atimTime, err := time.Parse(statTimeFormat, value)
		if err != nil {
			return fmt.Errorf("unable to parse atim: %s: %w", value, err)
		}
		stat_t.Atim = host.Timespec{
			Sec:  atimTime.Unix(),
			Nsec: atimTime.UnixNano() % 1e9,
		}
		return nil
	}),
	// time of last data modification, human-readable
	NewStatField("%y", func(value string, stat_t *host.Stat_t) error {
		var err error
		mtimTime, err := time.Parse(statTimeFormat, value)
		if err != nil {
			return fmt.Errorf("unable to parse mtim: %s: %w", value, err)
		}
		stat_t.Mtim = host.Timespec{
			Sec:  mtimTime.Unix(),
			Nsec: mtimTime.UnixNano() % 1e9,
		}
		return nil
	}),
	// time of last status change, human-readable
	NewStatField("%z", func(value string, stat_t *host.Stat_t) error {
		var err error
		ctimTime, err := time.Parse(statTimeFormat, value)
		if err != nil {
			return fmt.Errorf("unable to parse ctim: %s: %w", value, err)
		}
		stat_t.Ctim = host.Timespec{
			Sec:  ctimTime.Unix(),
			Nsec: ctimTime.UnixNano() % 1e9,
		}
		return nil
	}),
}

type baseHostRunReadFileRunRet struct {
	WaitStatus     host.WaitStatus
	WaitStatusErr  error
	StdoutCloseErr error
	StderrBuffer   bytes.Buffer
}

type baseHostRunReadFileReadCloser struct {
	Data     []byte
	Cmd      *host.Cmd
	Stdout   io.ReadCloser
	RunRetCh chan *baseHostRunReadFileRunRet
}

func (rc *baseHostRunReadFileReadCloser) Read(p []byte) (n int, err error) {
	if len(rc.Data) > 0 {
		n := copy(p, rc.Data)
		if n < len(rc.Data) {
			rc.Data = rc.Data[n:]
		} else {
			rc.Data = nil
		}
		return n, nil
	}
	n, err = rc.Stdout.Read(p)
	return n, err
}

func (rc *baseHostRunReadFileReadCloser) Close() error {
	var err error

	if closeErr := rc.Stdout.Close(); closeErr != nil {
		err = errors.Join(err, closeErr)
	}

	runRet := <-rc.RunRetCh
	if runRet.WaitStatusErr != nil {
		err = errors.Join(err, runRet.WaitStatusErr)
	} else {
		if !runRet.WaitStatus.Success() {
			if strings.Contains(runRet.StderrBuffer.String(), "Permission denied") {
				err = errors.Join(err, os.ErrPermission)
			} else if strings.Contains(runRet.StderrBuffer.String(), "No such file or directory") {
				err = errors.Join(err, os.ErrNotExist)
			} else {
				err = errors.Join(err, fmt.Errorf(
					"failed to run %s: %s\nstderr:\n%s",
					rc.Cmd, runRet.WaitStatus.String(), runRet.StderrBuffer.String(),
				))
			}
		}
	}

	if runRet.StdoutCloseErr != nil {
		err = errors.Join(err, runRet.StdoutCloseErr)
	}

	return err
}

// This partially implements host.Host interface, with the exception of the following functions:
// Run, String and Close. Full implementtations of the host.Host interface can embed this struct,
// and just implement the remaining methods.
// The use case for this is for share code across host.Host implementations that solely rely
// on spawning commands via Run.
type BaseHostRun struct {
	Host host.Host
}

func (h BaseHostRun) Geteuid(ctx context.Context) (uint64, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Geteuid")

	cmd := host.Cmd{
		Path: "id",
		Args: []string{"-u"},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
	if err != nil {
		return 0, err
	}
	if !waitStatus.Success() {
		return 0, fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}

	var uid uint64
	uid, err = strconv.ParseUint(strings.TrimSuffix(stdout, "\n"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse uid: %w", err)
	}

	return uid, nil
}

func (h BaseHostRun) Getegid(ctx context.Context) (uint64, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Getegid")

	cmd := host.Cmd{
		Path: "id",
		Args: []string{"-g"},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
	if err != nil {
		return 0, err
	}
	if !waitStatus.Success() {
		return 0, fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}

	var gid uint64
	gid, err = strconv.ParseUint(strings.TrimSuffix(stdout, "\n"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse gid: %w", err)
	}

	return gid, nil
}

func (h BaseHostRun) Chmod(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Chmod", "name", name, "mode", mode)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Chmod",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	cmd := host.Cmd{
		Path: "chmod",
		Args: []string{fmt.Sprintf("%o", mode), name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
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

func (h BaseHostRun) Chown(ctx context.Context, name string, uid, gid uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Chown", "name", name, "uid", uid, "gid", gid)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Chown",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	cmd := host.Cmd{
		Path: "chown",
		Args: []string{fmt.Sprintf("%d.%d", uid, gid), name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
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

func (h BaseHostRun) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lookup", "username", username)

	cmd := host.Cmd{
		Path: "cat",
		Args: []string{"/etc/passwd"},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
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

func (h BaseHostRun) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("LookupGroup", "name", name)

	cmd := host.Cmd{
		Path: "cat",
		Args: []string{"/etc/group"},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
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

func (h BaseHostRun) Lstat(ctx context.Context, name string) (*host.Stat_t, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lstat", "name", name)

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "Lstat",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	var format string
	for i, field := range statFields {
		if i > 0 {
			format += ","
		}
		format += field.format
	}

	cmd := host.Cmd{
		Path: "stat",
		Args: []string{
			fmt.Sprintf("--format=%s", format),
			name,
		},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
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

	values := strings.Split(strings.TrimRight(stdout, "\n"), ",")
	if len(values) != len(statFields) {
		return nil, fmt.Errorf("unable to parse stat output: %#v", stdout)
	}

	var stat_t host.Stat_t
	for i, value := range values {
		err := statFields[i].fn(value, &stat_t)
		if err != nil {
			return nil, fmt.Errorf("unable to parse stat %#v output: %w", statFields[i].format, err)
		}
	}

	return &stat_t, nil
}

func (h BaseHostRun) getFindFileType(fileType rune) (uint8, error) {
	switch fileType {
	case 's':
		return syscall.DT_SOCK, nil
	case 'l':
		return syscall.DT_LNK, nil
	case 'f':
		return syscall.DT_REG, nil
	case 'b':
		return syscall.DT_BLK, nil
	case 'd':
		return syscall.DT_DIR, nil
	case 'c':
		return syscall.DT_CHR, nil
	case 'p':
		return syscall.DT_FIFO, nil
	default:
		return 0, fmt.Errorf("unexpected file type from find: %c", fileType)
	}
}

func (h BaseHostRun) ReadDir(ctx context.Context, name string) ([]host.DirEnt, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("ReadDir", "name", name)

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "ReadDir",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	name = filepath.Clean(name)

	cmd := host.Cmd{
		Path: "find",
		Args: []string{name, "-maxdepth", "1", "-printf", "%i %y %p\\0"},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
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

	values := strings.Split(stdout, "\000")

	dirEnts := []host.DirEnt{}

	for _, value := range values[:len(values)-1] {
		dirEnt := host.DirEnt{}
		var fileType rune
		n, err := fmt.Sscanf(value, "%d %c %s", &dirEnt.Ino, &fileType, &dirEnt.Name)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to scan find output: %#v: %w", value, err)
		}
		if n != 3 {
			return nil, fmt.Errorf("failed to scan find output: %#v", value)
		}

		if filepath.Clean(dirEnt.Name) == name {
			continue
		}

		dirEnt.Name = filepath.Base(dirEnt.Name)

		dirEnt.Type, err = h.getFindFileType(fileType)
		if err != nil {
			return nil, err
		}

		dirEnts = append(dirEnts, dirEnt)
	}

	return dirEnts, nil
}

func (h BaseHostRun) Mkdir(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Mkdir", "name", name, "mode", mode)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Mkdir",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	cmd := host.Cmd{
		Path: "mkdir",
		Args: []string{name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
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

	return h.Chmod(ctx, name, mode)
}

func (h BaseHostRun) ReadFile(ctx context.Context, name string) (io.ReadCloser, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("ReadFile", "name", name)

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "ReadFile",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	stdoutReader, stdoutWriter := io.Pipe()
	stderrBuffer := bytes.Buffer{}
	cmd := host.Cmd{
		Path:   "cat",
		Args:   []string{name},
		Stdout: stdoutWriter,
		Stderr: &stderrBuffer,
	}

	runRetCh := make(chan *baseHostRunReadFileRunRet)

	go func() {
		waitStatus, waitStatusErr := h.Host.Run(ctx, cmd)
		closeErr := stdoutWriter.Close()
		runRetCh <- &baseHostRunReadFileRunRet{
			WaitStatus:     waitStatus,
			WaitStatusErr:  waitStatusErr,
			StdoutCloseErr: closeErr,
			StderrBuffer:   stderrBuffer,
		}
	}()

	readCloser := &baseHostRunReadFileReadCloser{
		Cmd:      &cmd,
		Stdout:   stdoutReader,
		RunRetCh: runRetCh,
	}

	// We require to read the first chunk of the stream here, as it enables to catch the various
	// errors we're expected to return.
	buff := make([]byte, 8192)
	n, err := stdoutReader.Read(buff)
	if err != nil {
		if readCloserErr := readCloser.Close(); readCloserErr != nil {
			err = errors.Join(err, readCloserErr)
		}
		return nil, err
	}

	readCloser.Data = buff[:n]

	return readCloser, nil
}

func (h BaseHostRun) Symlink(ctx context.Context, oldname, newname string) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Symlink", "oldname", oldname, "newname", newname)

	if !path.IsAbs(newname) {
		return &fs.PathError{
			Op:   "Symlink",
			Path: newname,
			Err:  errors.New("path must be absolute"),
		}
	}

	cmd := host.Cmd{
		Path: "ln",
		Args: []string{"-s", oldname, newname},
	}

	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
	if err != nil {
		return err
	}

	if !waitStatus.Success() {
		if strings.Contains(stderr, "Permission denied") {
			return os.ErrPermission
		}
		if strings.Contains(stderr, "No such file or directory") {
			return os.ErrNotExist
		}
		if strings.Contains(stderr, "File exists") {
			return os.ErrExist
		}
		return fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}

	return nil
}

func (h BaseHostRun) Readlink(ctx context.Context, name string) (string, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Readlink", "name", name)

	if !filepath.IsAbs(name) {
		return "", &fs.PathError{
			Op:   "Readlink",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	cmd := host.Cmd{
		Path: "readlink",
		Args: []string{"-vn", name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
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

func (h BaseHostRun) rmdir(ctx context.Context, name string) error {
	cmd := host.Cmd{
		Path: "rmdir",
		Args: []string{name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
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

func (h BaseHostRun) Remove(ctx context.Context, name string) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Remove", "name", name)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Remove",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	cmd := host.Cmd{
		Path: "rm",
		Args: []string{name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		if strings.Contains(stderr, "Is a directory") {
			return h.rmdir(ctx, name)
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

func (h BaseHostRun) WriteFile(ctx context.Context, name string, data []byte, mode uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("WriteFile", "name", name, "data", data, "mode", mode)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "WriteFile",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	cmd := host.Cmd{
		Path:  "sh",
		Args:  []string{"-c", fmt.Sprintf("cat > %s", shellescape.Quote(name))},
		Stdin: bytes.NewReader(data),
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, h.Host, cmd)
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

	return h.Chmod(ctx, name, mode)
}
