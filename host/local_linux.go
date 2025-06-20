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
)

// Local interacts with the local machine running the code.
type Local struct{}

func (h Local) Geteuid(ctx context.Context) (uint64, error) {
	return uint64(syscall.Geteuid()), nil
}

func (h Local) Getegid(ctx context.Context) (uint64, error) {
	return uint64(syscall.Getegid()), nil
}

func (h Local) getPathError(op, path string, err error) error {
	if err == nil {
		return nil
	}
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) {
		return &fs.PathError{
			Op:   op,
			Path: path,
			Err:  err,
		}
	}
	return err
}

func (h Local) Chmod(ctx context.Context, name string, mode types.FileMode) error {
	if !filepath.IsAbs(name) {
		return h.getPathError("Chmod", name, errors.New("path must be absolute"))
	}

	return h.getPathError("Chmod", name, syscall.Chmod(name, uint32(mode)))
}

func (h Local) Lchown(ctx context.Context, name string, uid, gid uint32) error {
	if !filepath.IsAbs(name) {
		return h.getPathError("Lchown", name, errors.New("path must be absolute"))
	}

	return h.getPathError("Lchown", name, syscall.Lchown(name, int(uid), int(gid)))
}

func (h Local) Lookup(ctx context.Context, username string) (*user.User, error) {
	return user.Lookup(username)
}

func (h Local) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	return user.LookupGroup(name)
}

func (h Local) Lstat(ctx context.Context, name string) (*types.Stat_t, error) {
	if !filepath.IsAbs(name) {
		return nil, h.getPathError("Lstat", name, errors.New("path must be absolute"))
	}

	var syscallStat_t syscall.Stat_t
	err := syscall.Lstat(name, &syscallStat_t)
	if err != nil {
		return nil, h.getPathError("Lstat", name, err)
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
	return lib.LocalReadDir(ctx, name)
}

func (h Local) Mkdir(ctx context.Context, name string, mode types.FileMode) error {
	if !filepath.IsAbs(name) {
		return h.getPathError("Mkdir", name, errors.New("path must be absolute"))
	}

	if err := syscall.Mkdir(name, uint32(mode)); err != nil {
		return h.getPathError("Mkdir", name, err)
	}
	return h.getPathError("Mkdir", name, syscall.Chmod(name, uint32(mode)))
}

func (h Local) ReadFile(ctx context.Context, name string) (io.ReadCloser, error) {
	if !filepath.IsAbs(name) {
		return nil, h.getPathError("ReadFile", name, errors.New("path must be absolute"))
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, h.getPathError("ReadFile", name, err)
	}
	return f, nil
}

func (h Local) Symlink(ctx context.Context, oldname, newname string) error {
	if !path.IsAbs(newname) {
		return h.getPathError("Symlink", newname, errors.New("path must be absolute"))
	}

	return h.getPathError("Symlink", newname, syscall.Symlink(oldname, newname))
}

func (h Local) Readlink(ctx context.Context, name string) (string, error) {
	if !filepath.IsAbs(name) {
		return "", h.getPathError("Readlink", name, errors.New("path must be absolute"))
	}

	oldname, err := os.Readlink(name)
	if err != nil {
		return "", h.getPathError("Readlink", name, err)
	}
	return oldname, nil
}

func (h Local) Remove(ctx context.Context, name string) error {
	if !filepath.IsAbs(name) {
		return h.getPathError("Remove", name, errors.New("path must be absolute"))
	}

	return h.getPathError("Remove", name, os.Remove(name))
}

func (h Local) Mknod(ctx context.Context, pathName string, mode types.FileMode, dev types.FileDevice) error {
	if !path.IsAbs(pathName) {
		return h.getPathError("Mknod", pathName, fmt.Errorf("path must be absolute"))
	}

	if dev != types.FileDevice(int(dev)) {
		return h.getPathError("Mknod", pathName, fmt.Errorf("dev value is too big: %#v", dev))
	}

	if err := syscall.Mknod(pathName, uint32(mode), int(dev)); err != nil {
		return h.getPathError("Mknod", pathName, err)
	}

	return h.getPathError("Mknod", pathName, syscall.Chmod(pathName, uint32(mode&types.FileModeBitsMask)))
}

func (h Local) Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
	return lib.LocalRun(ctx, cmd)
}

func (h Local) WriteFile(ctx context.Context, name string, data io.Reader, mode types.FileMode) error {
	if !filepath.IsAbs(name) {
		return h.getPathError("WriteFile", name, errors.New("path must be absolute"))
	}

	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(mode))
	if err != nil {
		return h.getPathError("WriteFile", name, err)
	}

	_, err = io.Copy(file, data)
	if err != nil {
		return h.getPathError("WriteFile", name, errors.Join(err, file.Close()))
	}

	return h.getPathError("WriteFile", name, errors.Join(syscall.Chmod(name, uint32(mode)), file.Close()))
}

func (h Local) AppendFile(ctx context.Context, name string, data io.Reader, mode types.FileMode) error {
	if !filepath.IsAbs(name) {
		return h.getPathError("AppendFile", name, errors.New("path must be absolute"))
	}

	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.FileMode(mode))
	if err != nil {
		return h.getPathError("AppendFile", name, err)
	}

	_, err = io.Copy(file, data)
	if err != nil {
		return h.getPathError("AppendFile", name, errors.Join(err, file.Close()))
	}

	return h.getPathError("AppendFile", name, errors.Join(syscall.Chmod(name, uint32(mode)), file.Close()))
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
