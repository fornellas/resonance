package host

import (
	"context"
	"errors"
	"os"
	"os/user"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host"
)

type cmdHostOnly struct {
	T    *testing.T
	Host host.Host
}

func (h cmdHostOnly) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	err := errors.New("unexpected call received: Chmod")
	h.T.Fatal(err)
	return err
}

func (h cmdHostOnly) Chown(ctx context.Context, name string, uid, gid int) error {
	err := errors.New("unexpected call received: Chown")
	h.T.Fatal(err)
	return err
}

func (h cmdHostOnly) Lookup(ctx context.Context, username string) (*user.User, error) {
	err := errors.New("unexpected call received: Lookup")
	h.T.Fatal(err)
	return nil, err
}

func (h cmdHostOnly) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	err := errors.New("unexpected call received: LookupGroup")
	h.T.Fatal(err)
	return nil, err
}

func (h cmdHostOnly) Lstat(ctx context.Context, name string) (*host.Stat_t, error) {
	err := errors.New("unexpected call received: Lstat")
	h.T.Fatal(err)
	return nil, err
}

func (h cmdHostOnly) Mkdir(ctx context.Context, name string, mode uint32) error {
	err := errors.New("unexpected call received: Mkdir")
	h.T.Fatal(err)
	return err
}

func (h cmdHostOnly) ReadFile(ctx context.Context, name string) ([]byte, error) {
	err := errors.New("unexpected call received: ReadFile")
	h.T.Fatal(err)
	return nil, err
}

func (h cmdHostOnly) Remove(ctx context.Context, name string) error {
	err := errors.New("unexpected call received: Remove")
	h.T.Fatal(err)
	return err
}

func (h cmdHostOnly) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	err := errors.New("unexpected call received: WriteFile")
	h.T.Fatal(err)
	return err
}

func (h cmdHostOnly) String() string {
	return h.Host.String()
}

func (h cmdHostOnly) Type() string {
	return h.Host.Type()
}

func (h cmdHostOnly) Close() error {
	return h.Host.Close()
}

type localRunOnly struct {
	cmdHostOnly
	Host host.Host
}

func (lro localRunOnly) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	return lro.Host.Run(ctx, cmd)
}

func newLocalRunOnly(t *testing.T, host host.Host) localRunOnly {
	run := localRunOnly{
		Host: host,
	}
	run.cmdHostOnly.T = t
	run.cmdHostOnly.Host = host
	return run
}

type runner struct {
	cmdHost
	Host host.Host
}

func (r runner) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	return r.Host.Run(ctx, cmd)
}

func (r runner) String() string {
	return r.Host.String()
}

func (r runner) Type() string {
	return r.Host.Type()
}

func (r runner) Close() error {
	return r.Host.Close()
}

func newRunner(host host.Host) runner {
	run := runner{
		Host: host,
	}
	run.cmdHost.Host = host
	return run
}

func TestCmdHost(t *testing.T) {
	host := newRunner(newLocalRunOnly(t, Local{}))
	testHost(t, host)
	defer func() { require.NoError(t, host.Close()) }()
	require.NoError(t, host.Close())
}
