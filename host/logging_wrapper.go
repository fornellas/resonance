package host

import (
	"context"
	"io"
	"os/user"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

// LoggingWrapper wraps Host logging received commands.
type LoggingWrapper struct {
	host types.Host
}

func NewLoggingWrapper(host types.Host) *LoggingWrapper {
	return &LoggingWrapper{host: host}
}

func (h *LoggingWrapper) Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Run", "cmd", cmd)
	return h.host.Run(ctx, cmd)
}

func (h *LoggingWrapper) String() string {
	return h.host.String()
}

func (h *LoggingWrapper) Type() string {
	return h.host.Type()
}

func (h *LoggingWrapper) Close(ctx context.Context) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Close")
	return h.host.Close(ctx)
}

func (h *LoggingWrapper) Geteuid(ctx context.Context) (uint64, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Geteuid")
	return h.host.Geteuid(ctx)
}

func (h *LoggingWrapper) Getegid(ctx context.Context) (uint64, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Getegid")
	return h.host.Getegid(ctx)
}

func (h *LoggingWrapper) Chmod(ctx context.Context, name string, mode types.FileMode) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Chmod", "name", name, "mode", mode)
	return h.host.Chmod(ctx, name, mode)
}

func (h *LoggingWrapper) Lchown(ctx context.Context, name string, uid, gid uint32) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Lchown", "name", name, "uid", uid, "gid", gid)
	return h.host.Lchown(ctx, name, uid, gid)
}

func (h *LoggingWrapper) Lookup(ctx context.Context, username string) (*user.User, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Lookup", "username", username)
	return h.host.Lookup(ctx, username)
}

func (h *LoggingWrapper) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("LookupGroup", "name", name)
	return h.host.LookupGroup(ctx, name)
}

func (h *LoggingWrapper) Lstat(ctx context.Context, name string) (*types.Stat_t, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Lstat", "name", name)
	return h.host.Lstat(ctx, name)
}

func (h *LoggingWrapper) ReadDir(ctx context.Context, name string) (dirEntResultCh <-chan types.DirEntResult, cancel func()) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("ReadDir", "name", name)
	return h.host.ReadDir(ctx, name)
}

func (h *LoggingWrapper) Mkdir(ctx context.Context, name string, mode types.FileMode) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Mkdir", "name", name, "mode", mode)
	return h.host.Mkdir(ctx, name, mode)
}

func (h *LoggingWrapper) ReadFile(ctx context.Context, name string) (io.ReadCloser, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("ReadFile", "name", name)
	return h.host.ReadFile(ctx, name)
}

func (h *LoggingWrapper) Symlink(ctx context.Context, oldname, newname string) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Symlink", "oldname", oldname, "newname", newname)
	return h.host.Symlink(ctx, oldname, newname)
}

func (h *LoggingWrapper) Readlink(ctx context.Context, name string) (string, error) {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Readlink", "name", name)
	return h.host.Readlink(ctx, name)
}

func (h *LoggingWrapper) Remove(ctx context.Context, name string) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Remove", "name", name)
	return h.host.Remove(ctx, name)
}

func (h *LoggingWrapper) Mknod(ctx context.Context, path string, mode types.FileMode, dev types.FileDevice) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("Mknod", "path", path, "mode", mode, "dev", dev)
	return h.host.Mknod(ctx, path, mode, dev)
}

func (h *LoggingWrapper) WriteFile(ctx context.Context, name string, data io.Reader, mode types.FileMode) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🖥️ Host", "type", h.host.Type(), "name", h.host.String())
	logger.Debug("WriteFile", "name", name, "mode", mode)
	return h.host.WriteFile(ctx, name, data, mode)
}
