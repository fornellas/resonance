package host

import (
	"context"
	"errors"
	"io"
	"os/user"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host"
)

// BaseHostNoCallsTest implements host.Host, forwarding calls to String(), Type() and Close() to an
// underlying host.Host, while failing calls to any other functions.
// The use casse for this is as a base Host implementation during tests.
type BaseHostNoCallsTest struct {
	T    *testing.T
	Host host.Host
}

func (h BaseHostNoCallsTest) Geteuid(ctx context.Context) (uint64, error) {
	err := errors.New("unexpected call received: Geteuid")
	h.T.Fatal(err)
	return 0, err
}

func (h BaseHostNoCallsTest) Getegid(ctx context.Context) (uint64, error) {
	err := errors.New("unexpected call received: Getegid")
	h.T.Fatal(err)
	return 0, err
}

func (h BaseHostNoCallsTest) Chmod(ctx context.Context, name string, mode uint32) error {
	err := errors.New("unexpected call received: Chmod")
	h.T.Fatal(err)
	return err
}

func (h BaseHostNoCallsTest) Chown(ctx context.Context, name string, uid, gid uint32) error {
	err := errors.New("unexpected call received: Chown")
	h.T.Fatal(err)
	return err
}

func (h BaseHostNoCallsTest) Lookup(ctx context.Context, username string) (*user.User, error) {
	err := errors.New("unexpected call received: Lookup")
	h.T.Fatal(err)
	return nil, err
}

func (h BaseHostNoCallsTest) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	err := errors.New("unexpected call received: LookupGroup")
	h.T.Fatal(err)
	return nil, err
}

func (h BaseHostNoCallsTest) Lstat(ctx context.Context, name string) (*host.Stat_t, error) {
	err := errors.New("unexpected call received: Lstat")
	h.T.Fatal(err)
	return nil, err
}

func (h BaseHostNoCallsTest) ReadDir(ctx context.Context, name string) ([]host.DirEnt, error) {
	err := errors.New("unexpected call received: ReadDir")
	h.T.Fatal(err)
	return nil, err
}

func (h BaseHostNoCallsTest) Mkdir(ctx context.Context, name string, mode uint32) error {
	err := errors.New("unexpected call received: Mkdir")
	h.T.Fatal(err)
	return err
}

func (h BaseHostNoCallsTest) ReadFile(ctx context.Context, name string) (io.ReadCloser, error) {
	err := errors.New("unexpected call received: ReadFile")
	h.T.Fatal(err)
	return nil, err
}

func (h BaseHostNoCallsTest) Symlink(ctx context.Context, oldname, newname string) error {
	err := errors.New("unexpected call received: Symlink")
	h.T.Fatal(err)
	return err
}

func (h BaseHostNoCallsTest) Readlink(ctx context.Context, name string) (string, error) {
	err := errors.New("unexpected call received: Readlink")
	h.T.Fatal(err)
	return "", err
}

func (h BaseHostNoCallsTest) Remove(ctx context.Context, name string) error {
	err := errors.New("unexpected call received: Remove")
	h.T.Fatal(err)
	return err
}

func (h BaseHostNoCallsTest) Mknod(ctx context.Context, pathName string, mode uint32, dev uint64) error {
	err := errors.New("unexpected call received: BaseHostNoCallsTest")
	h.T.Fatal(err)
	return err
}

func (h BaseHostNoCallsTest) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	err := errors.New("unexpected call received: Run")
	h.T.Fatal(err)
	return host.WaitStatus{}, err
}

func (h BaseHostNoCallsTest) WriteFile(ctx context.Context, name string, data io.Reader, mode uint32) error {
	err := errors.New("unexpected call received: WriteFile")
	h.T.Fatal(err)
	return err
}

func (h BaseHostNoCallsTest) String() string {
	return h.Host.String()
}

func (h BaseHostNoCallsTest) Type() string {
	return h.Host.Type()
}

func (h BaseHostNoCallsTest) Close(ctx context.Context) error {
	return h.Host.Close(ctx)
}

// HostLocalRunOnlyTest only allows running Local.Run, failing other Host.
type HostLocalRunOnlyTest struct {
	BaseHostNoCallsTest
	Host host.Host
}

func (h HostLocalRunOnlyTest) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	return h.Host.Run(ctx, cmd)
}

func NewHostLocalRunOnlyTest(t *testing.T) HostLocalRunOnlyTest {
	localHost := Local{}
	h := HostLocalRunOnlyTest{
		Host: localHost,
	}
	h.BaseHostNoCallsTest.T = t
	h.BaseHostNoCallsTest.Host = localHost
	return h
}

// BaseHostRunTester helps test BaseHostRun
type BaseHostRunTester struct {
	BaseHostRun
	Host host.Host
}

func NewBaseHostRunTester(host host.Host) BaseHostRunTester {
	run := BaseHostRunTester{
		Host: host,
	}
	run.BaseHostRun.Host = host
	return run
}

func (r BaseHostRunTester) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	return r.Host.Run(ctx, cmd)
}

func (r BaseHostRunTester) String() string {
	return r.Host.String()
}

func (r BaseHostRunTester) Type() string {
	return r.Host.Type()
}

func (r BaseHostRunTester) Close(ctx context.Context) error {
	return r.Host.Close(ctx)
}

func TestBaseHostRun(t *testing.T) {
	host := NewBaseHostRunTester(NewHostLocalRunOnlyTest(t))
	testHost(t, host)
	ctx := context.Background()
	defer func() { require.NoError(t, host.Close(ctx)) }()
	require.NoError(t, host.Close(ctx))
}
