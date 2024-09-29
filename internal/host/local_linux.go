package host

import (
	"context"
	"os"
	"os/user"
	"syscall"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/internal/host/local_run"
	"github.com/fornellas/resonance/log"
)

// Local interacts with the local machine running the code.
type Local struct{}

func (l Local) Chmod(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Chmod", "name", name, "mode", mode)
	return syscall.Chmod(name, mode)
}

func (l Local) Chown(ctx context.Context, name string, uid, gid int) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Chown", "name", name, "uid", uid, "gid", gid)
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

func (l Local) Lstat(ctx context.Context, name string) (*syscall.Stat_t, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Lstat", "name", name)

	var stat_t syscall.Stat_t
	err := syscall.Lstat(name, &stat_t)
	if err != nil {
		return nil, err
	}
	return &stat_t, nil
}

func (l Local) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Mkdir", "name", name, "perm", perm)
	return os.Mkdir(name, perm)
}

func (l Local) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("ReadFile", "name", name)
	return os.ReadFile(name)
}

func (l Local) Remove(ctx context.Context, name string) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Remove", "name", name)
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
