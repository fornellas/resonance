package host

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/internal/host/local_run"
	"github.com/fornellas/resonance/log"
)

// Local interacts with the local machine running the code.
type Local struct{}

func (l Local) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Chmod", "name", name, "mode", mode)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Chmod",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	return os.Chmod(name, mode)
}

func (l Local) Chown(ctx context.Context, name string, uid, gid int) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Chown", "name", name, "uid", uid, "gid", gid)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Chown",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	return os.Chown(name, uid, gid)
}

func (l Local) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Lookup", "username", username)
	return user.Lookup(username)
}

func (l Local) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("LookupGroup", "name", name)
	return user.LookupGroup(name)
}

func (l Local) Lstat(ctx context.Context, name string) (*host.Stat_t, error) {
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

	return &host.Stat_t{
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
		Atim: host.Timespec{
			Sec:  int64(syscallStat_t.Atim.Sec),
			Nsec: int64(syscallStat_t.Atim.Nsec),
		},
		Mtim: host.Timespec{
			Sec:  int64(syscallStat_t.Mtim.Sec),
			Nsec: int64(syscallStat_t.Mtim.Nsec),
		},
		Ctim: host.Timespec{
			Sec:  int64(syscallStat_t.Ctim.Sec),
			Nsec: int64(syscallStat_t.Ctim.Nsec),
		},
	}, nil
}

func (l Local) Mkdir(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Mkdir", "name", name, "mode", mode)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Mkdir",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	if err := syscall.Mkdir(name, mode); err != nil {
		return err
	}
	return syscall.Chmod(name, mode)
}

func (l Local) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("ReadFile", "name", name)

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "ReadFile",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	return os.ReadFile(name)
}

func (l Local) Remove(ctx context.Context, name string) error {
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

func (l Local) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Run", "cmd", cmd)
	return local_run.Run(ctx, cmd)
}

func (l Local) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	logger := log.MustLogger(ctx)
	logger.Debug("WriteFile", "name", name, "data", data, "perm", perm)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "WriteFile",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	return os.WriteFile(name, data, perm)
}

func (l Local) String() string {
	return "localhost"
}

func (l Local) Type() string {
	return "localhost"
}

func (l Local) Close() error {
	return nil
}
