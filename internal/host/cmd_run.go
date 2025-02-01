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

type CmdHostReadFileRunRet struct {
	WaitStatus     host.WaitStatus
	WaitStatusErr  error
	StdoutCloseErr error
	StderrBuffer   bytes.Buffer
}

type CmdHostReadFileReadCloser struct {
	Data     []byte
	Cmd      *host.Cmd
	Stdout   io.ReadCloser
	RunRetCh chan *CmdHostReadFileRunRet
}

func (c *CmdHostReadFileReadCloser) Read(p []byte) (n int, err error) {
	if len(c.Data) > 0 {
		n := copy(p, c.Data)
		if n < len(c.Data) {
			c.Data = c.Data[n:]
		} else {
			c.Data = nil
		}
		return n, nil
	}
	n, err = c.Stdout.Read(p)
	return n, err
}

func (c *CmdHostReadFileReadCloser) Close() error {
	var err error

	if closeErr := c.Stdout.Close(); closeErr != nil {
		err = errors.Join(err, closeErr)
	}

	runRet := <-c.RunRetCh
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
					c.Cmd, runRet.WaitStatus.String(), runRet.StderrBuffer.String(),
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
type cmdHost struct {
	Host host.Host
}

func (br cmdHost) Geteuid(ctx context.Context) (uint64, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Geteuid")

	cmd := host.Cmd{
		Path: "id",
		Args: []string{"-u"},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

func (br cmdHost) Getegid(ctx context.Context) (uint64, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Getegid")

	cmd := host.Cmd{
		Path: "id",
		Args: []string{"-g"},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

func (br cmdHost) Chmod(ctx context.Context, name string, mode uint32) error {
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
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

func (br cmdHost) Chown(ctx context.Context, name string, uid, gid uint32) error {
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
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

func (br cmdHost) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lookup", "username", username)

	cmd := host.Cmd{
		Path: "cat",
		Args: []string{"/etc/passwd"},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

func (br cmdHost) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("LookupGroup", "name", name)

	cmd := host.Cmd{
		Path: "cat",
		Args: []string{"/etc/group"},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

type statField struct {
	format string
	fn     func(string, *host.Stat_t) error
}

func newStatField(format string, fn func(value string, stat_t *host.Stat_t) error) statField {
	return statField{format: format, fn: fn}
}

var statTimeFormat = "2006-01-02 15:04:05.999999999 -0700"

var statFields = []statField{
	// device number in decimal (st_dev)
	newStatField("%d", func(value string, stat_t *host.Stat_t) error {
		var err error
		stat_t.Dev, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse dev: %s", value)
		}
		return nil
	}),
	// inode number
	newStatField("%i", func(value string, stat_t *host.Stat_t) error {
		var err error
		stat_t.Ino, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse ino: %s", value)
		}
		return nil
	}),
	// number of hard links
	newStatField("%h", func(value string, stat_t *host.Stat_t) error {
		var err error
		stat_t.Nlink, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse nlink: %s", value)
		}
		return nil
	}),
	// raw mode in hex
	newStatField("%f", func(value string, stat_t *host.Stat_t) error {
		mode64, err := strconv.ParseUint(value, 16, 32)
		if err != nil {
			return fmt.Errorf("unable to parse mode: %s", value)
		}
		stat_t.Mode = uint32(mode64)
		return nil
	}),
	// user ID of owner
	newStatField("%u", func(value string, stat_t *host.Stat_t) error {
		uid64, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("unable to parse uid: %s", value)
		}
		stat_t.Uid = uint32(uid64)
		return nil
	}),
	// group ID of owner
	newStatField("%g", func(value string, stat_t *host.Stat_t) error {
		gid64, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("unable to parse gid: %s", value)
		}
		stat_t.Gid = uint32(gid64)
		return nil
	}),
	// device type in decimal (st_rdev)
	newStatField("%r", func(value string, stat_t *host.Stat_t) error {
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
	newStatField("%s", func(value string, stat_t *host.Stat_t) error {
		var err error
		stat_t.Size, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse size: %s", value)
		}
		return nil
	}),
	// the size in bytes of each block reported by %b
	newStatField("%B", func(value string, stat_t *host.Stat_t) error {
		var err error
		stat_t.Blksize, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse blksize: %s", value)
		}
		return nil
	}),
	// number of blocks allocated (see %B)
	newStatField("%b", func(value string, stat_t *host.Stat_t) error {
		var err error
		stat_t.Blocks, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse blocks: %s", value)
		}
		return nil
	}),
	// time of last access, human-readable
	newStatField("%x", func(value string, stat_t *host.Stat_t) error {
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
	newStatField("%y", func(value string, stat_t *host.Stat_t) error {
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
	newStatField("%z", func(value string, stat_t *host.Stat_t) error {
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

func (br cmdHost) Lstat(ctx context.Context, name string) (*host.Stat_t, error) {
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
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

func getFindFileType(fileType rune) (uint8, error) {
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

func (br cmdHost) ReadDir(ctx context.Context, name string) ([]host.DirEnt, error) {
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
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

		dirEnt.Type, err = getFindFileType(fileType)
		if err != nil {
			return nil, err
		}

		dirEnts = append(dirEnts, dirEnt)
	}

	return dirEnts, nil
}

func (br cmdHost) Mkdir(ctx context.Context, name string, mode uint32) error {
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
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

	return br.Chmod(ctx, name, mode)
}

func (br cmdHost) ReadFile(ctx context.Context, name string) (io.ReadCloser, error) {
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

	runRetCh := make(chan *CmdHostReadFileRunRet)

	go func() {
		waitStatus, waitStatusErr := br.Host.Run(ctx, cmd)
		closeErr := stdoutWriter.Close()
		runRetCh <- &CmdHostReadFileRunRet{
			WaitStatus:     waitStatus,
			WaitStatusErr:  waitStatusErr,
			StdoutCloseErr: closeErr,
			StderrBuffer:   stderrBuffer,
		}
	}()

	readCloser := &CmdHostReadFileReadCloser{
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

func (br cmdHost) Symlink(ctx context.Context, oldname, newname string) error {
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

	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

func (br cmdHost) Readlink(ctx context.Context, name string) (string, error) {
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
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

func (br cmdHost) rmdir(ctx context.Context, name string) error {
	cmd := host.Cmd{
		Path: "rmdir",
		Args: []string{name},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

func (br cmdHost) Remove(ctx context.Context, name string) error {
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
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		if strings.Contains(stderr, "Is a directory") {
			return br.rmdir(ctx, name)
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

func (br cmdHost) WriteFile(ctx context.Context, name string, data []byte, mode uint32) error {
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
	waitStatus, stdout, stderr, err := host.Run(ctx, br.Host, cmd)
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

	return br.Chmod(ctx, name, mode)
}
