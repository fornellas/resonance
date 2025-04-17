package host

import (
	"context"
	"io"
	"os/user"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

// LoggingHost wraps Host logging received commands.
type LoggingHost struct {
	host types.Host
}

func NewLoggingHost(host types.Host) types.Host {
	return &LoggingHost{host: host}
}

func (h *LoggingHost) Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Run", "cmd", cmd)
	return h.host.Run(ctx, cmd)
}

func (h *LoggingHost) String() string {
	return h.host.String()
}

func (h *LoggingHost) Type() string {
	return h.host.Type()
}

func (h *LoggingHost) Close(ctx context.Context) error {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Close")
	return h.host.Close(ctx)
}

func (h *LoggingHost) Geteuid(ctx context.Context) (uint64, error) {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Geteuid")
	return h.host.Geteuid(ctx)
}

func (h *LoggingHost) Getegid(ctx context.Context) (uint64, error) {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Getegid")
	return h.host.Getegid(ctx)
}

func (h *LoggingHost) Chmod(ctx context.Context, name string, mode types.FileMode) error {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Chmod", "name", name, "mode", mode)
	return h.host.Chmod(ctx, name, mode)
}

func (h *LoggingHost) Lchown(ctx context.Context, name string, uid, gid uint32) error {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Lchown", "name", name, "uid", uid, "gid", gid)
	return h.host.Lchown(ctx, name, uid, gid)
}

func (h *LoggingHost) Lookup(ctx context.Context, username string) (*user.User, error) {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Lookup", "username", username)
	return h.host.Lookup(ctx, username)
}

func (h *LoggingHost) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("LookupGroup", "name", name)
	return h.host.LookupGroup(ctx, name)
}

func (h *LoggingHost) Lstat(ctx context.Context, name string) (*types.Stat_t, error) {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Lstat", "name", name)
	return h.host.Lstat(ctx, name)
}

func (h *LoggingHost) ReadDir(ctx context.Context, name string) (dirEntResultCh <-chan types.DirEntResult, cancel func()) {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("ReadDir", "name", name)
	return h.host.ReadDir(ctx, name)
}

func (h *LoggingHost) Mkdir(ctx context.Context, name string, mode types.FileMode) error {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Mkdir", "name", name, "mode", mode)
	return h.host.Mkdir(ctx, name, mode)
}

func (h *LoggingHost) ReadFile(ctx context.Context, name string) (io.ReadCloser, error) {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("ReadFile", "name", name)
	return h.host.ReadFile(ctx, name)
}

func (h *LoggingHost) Symlink(ctx context.Context, oldname, newname string) error {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Symlink", "oldname", oldname, "newname", newname)
	return h.host.Symlink(ctx, oldname, newname)
}

func (h *LoggingHost) Readlink(ctx context.Context, name string) (string, error) {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Readlink", "name", name)
	return h.host.Readlink(ctx, name)
}

func (h *LoggingHost) Remove(ctx context.Context, name string) error {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Remove", "name", name)
	return h.host.Remove(ctx, name)
}

func (h *LoggingHost) Mknod(ctx context.Context, path string, mode types.FileMode, dev types.FileDevice) error {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("Mknod", "path", path, "mode", mode, "dev", dev)
	return h.host.Mknod(ctx, path, mode, dev)
}

func (h *LoggingHost) WriteFile(ctx context.Context, name string, data io.Reader, mode types.FileMode) error {
	ctx, logger := log.WithGroupAttrs(ctx, "🖥️ Host")
	logger.Debug("WriteFile", "name", name, "mode", mode)
	return h.host.WriteFile(ctx, name, data, mode)
}
