package host

import (
	"context"
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
	return os.Chmod(name, mode)
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

func (l Local) Lstat(ctx context.Context, name string) (host.HostFileInfo, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Lstat", "name", name)
	fileInfo, err := os.Lstat(name)
	if err != nil {
		return host.HostFileInfo{}, err
	}
	stat_t := fileInfo.Sys().(*syscall.Stat_t)
	return host.HostFileInfo{
		Name:    filepath.Base(name),
		Size:    fileInfo.Size(),
		Mode:    fileInfo.Mode(),
		ModTime: fileInfo.ModTime(),
		IsDir:   fileInfo.IsDir(),
		Uid:     stat_t.Uid,
		Gid:     stat_t.Gid,
	}, nil
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

func (l Local) Close() error {
	return nil
}
