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

type baseRunOnly struct {
	T    *testing.T
	Host host.Host
}

func (bro baseRunOnly) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	err := errors.New("unexpected call received: Chmod")
	bro.T.Fatal(err)
	return err
}

func (bro baseRunOnly) Chown(ctx context.Context, name string, uid, gid int) error {
	err := errors.New("unexpected call received: Chown")
	bro.T.Fatal(err)
	return err
}

func (bro baseRunOnly) Lookup(ctx context.Context, username string) (*user.User, error) {
	err := errors.New("unexpected call received: Lookup")
	bro.T.Fatal(err)
	return nil, err
}

func (bro baseRunOnly) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	err := errors.New("unexpected call received: LookupGroup")
	bro.T.Fatal(err)
	return nil, err
}

func (bro baseRunOnly) Lstat(ctx context.Context, name string) (host.HostFileInfo, error) {
	err := errors.New("unexpected call received: Lstat")
	bro.T.Fatal(err)
	return host.HostFileInfo{}, err
}

func (bro baseRunOnly) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	err := errors.New("unexpected call received: Mkdir")
	bro.T.Fatal(err)
	return err
}

func (bro baseRunOnly) ReadFile(ctx context.Context, name string) ([]byte, error) {
	err := errors.New("unexpected call received: ReadFile")
	bro.T.Fatal(err)
	return nil, err
}

func (bro baseRunOnly) Remove(ctx context.Context, name string) error {
	err := errors.New("unexpected call received: Remove")
	bro.T.Fatal(err)
	return err
}

func (bro baseRunOnly) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	err := errors.New("unexpected call received: WriteFile")
	bro.T.Fatal(err)
	return err
}

func (bro baseRunOnly) String() string {
	return bro.Host.String()
}

func (bro baseRunOnly) Close() error {
	return bro.Host.Close()
}

type localRunOnly struct {
	baseRunOnly
	Host host.Host
}

func (lro localRunOnly) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	return lro.Host.Run(ctx, cmd)
}

func newLocalRunOnly(t *testing.T, host host.Host) localRunOnly {
	run := localRunOnly{
		Host: host,
	}
	run.baseRunOnly.T = t
	run.baseRunOnly.Host = host
	return run
}

type runner struct {
	baseRun
	Host host.Host
}

func (r runner) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	return r.Host.Run(ctx, cmd)
}

func (r runner) String() string {
	return r.Host.String()
}

func (r runner) Close() error {
	return r.Host.Close()
}

func newRunner(host host.Host) runner {
	run := runner{
		Host: host,
	}
	run.baseRun.Host = host
	return run
}

func TestBaseRun(t *testing.T) {
	host := newRunner(newLocalRunOnly(t, Local{}))
	testHost(t, host)
	defer func() { require.NoError(t, host.Close()) }()
	require.NoError(t, host.Close())
}
