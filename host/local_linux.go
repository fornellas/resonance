package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"syscall"

	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

// Local interacts with the local machine running the code.
type Local struct{}

func (h Local) Geteuid(ctx context.Context) (uint64, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Geteuid")

	return uint64(syscall.Geteuid()), nil
}

func (h Local) Getegid(ctx context.Context) (uint64, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Getegid")

	return uint64(syscall.Getegid()), nil
}

func (h Local) Chmod(ctx context.Context, name string, mode types.FileMode) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Chmod", "name", name, "mode", mode)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Chmod",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	return syscall.Chmod(name, uint32(mode))
}

func (h Local) Lchown(ctx context.Context, name string, uid, gid uint32) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Lchown", "name", name, "uid", uid, "gid", gid)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Chown",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	return syscall.Lchown(name, int(uid), int(gid))
}

func (h Local) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Lookup", "username", username)
	return user.Lookup(username)
}

func (h Local) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("LookupGroup", "name", name)
	return user.LookupGroup(name)
}

func (h Local) Lstat(ctx context.Context, name string) (*types.Stat_t, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Lstat", "name", name)

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "Lstat",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	var syscallStat_t syscall.Stat_t
	err := syscall.Lstat(name, &syscallStat_t)
	if err != nil {
		return nil, err
	}

	return &types.Stat_t{
		Dev:     syscallStat_t.Dev,
		Ino:     syscallStat_t.Ino,
		Nlink:   uint64(syscallStat_t.Nlink),
		Mode:    syscallStat_t.Mode,
		Uid:     syscallStat_t.Uid,
		Gid:     syscallStat_t.Gid,
		Rdev:    syscallStat_t.Rdev,
		Size:    syscallStat_t.Size,
		Blksize: int64(syscallStat_t.Blksize),
		Blocks:  syscallStat_t.Blocks,
		Atim: types.Timespec{
			Sec:  int64(syscallStat_t.Atim.Sec),
			Nsec: int64(syscallStat_t.Atim.Nsec),
		},
		Mtim: types.Timespec{
			Sec:  int64(syscallStat_t.Mtim.Sec),
			Nsec: int64(syscallStat_t.Mtim.Nsec),
		},
		Ctim: types.Timespec{
			Sec:  int64(syscallStat_t.Ctim.Sec),
			Nsec: int64(syscallStat_t.Ctim.Nsec),
		},
	}, nil
}

func (h Local) ReadDir(ctx context.Context, name string) (<-chan types.DirEntResult, func()) {
	logger := log.MustLogger(ctx)
	logger.Debug("ReadDir", "name", name)

	return lib.LocalReadDir(ctx, name)
}

func (h Local) Mkdir(ctx context.Context, name string, mode types.FileMode) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Mkdir", "name", name, "mode", mode)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Mkdir",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	if err := syscall.Mkdir(name, uint32(mode)); err != nil {
		return err
	}
	return syscall.Chmod(name, uint32(mode))
}

func (h Local) ReadFile(ctx context.Context, name string) (io.ReadCloser, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("ReadFile", "name", name)

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "ReadFile",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (h Local) Symlink(ctx context.Context, oldname, newname string) error {
	if !path.IsAbs(newname) {
		return &fs.PathError{
			Op:   "Symlink",
			Path: newname,
			Err:  errors.New("path must be absolute"),
		}
	}

	return syscall.Symlink(oldname, newname)
}

func (h Local) Readlink(ctx context.Context, name string) (string, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Readlink", "name", name)

	if !filepath.IsAbs(name) {
		return "", &fs.PathError{
			Op:   "Readlink",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	return os.Readlink(name)
}

func (h Local) Remove(ctx context.Context, name string) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Remove", "name", name)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Remove",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	return os.Remove(name)
}

func (h Local) Mknod(ctx context.Context, pathName string, mode types.FileMode, dev types.FileDevice) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Mknod", "pathName", pathName, "mode", mode, "dev", dev)

	if !path.IsAbs(pathName) {
		return fmt.Errorf("path must be absolute: %#v", pathName)
	}

	if dev != types.FileDevice(int(dev)) {
		return fmt.Errorf("dev value is too big: %#v", dev)
	}

	if err := syscall.Mknod(pathName, uint32(mode), int(dev)); err != nil {
		return err
	}

	return syscall.Chmod(pathName, uint32(mode)&07777)
}

func (h Local) Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Run", "cmd", cmd)
	return lib.LocalRun(ctx, cmd)
}

func (h Local) WriteFile(ctx context.Context, name string, data io.Reader, mode types.FileMode) error {
	logger := log.MustLogger(ctx)
	logger.Debug("WriteFile", "name", name, "data", data, "mode", mode)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "WriteFile",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(mode))
	if err != nil {
		return err
	}

	_, err = io.Copy(file, data)
	if err != nil {
		return errors.Join(err, file.Close())
	}

	return errors.Join(syscall.Chmod(name, uint32(mode)), file.Close())
}

func (h Local) String() string {
	return "localhost"
}

func (h Local) Type() string {
	return "localhost"
}

func (h Local) Close(ctx context.Context) error {
	return nil
}
