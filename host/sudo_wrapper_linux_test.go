package host

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

type HostLocalRunSudoOnlyTest struct {
	T        *testing.T
	BaseHost types.BaseHost
}

func (h HostLocalRunSudoOnlyTest) Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
	if cmd.Path != "sudo" {
		err := fmt.Errorf("attempted to run non-sudo command: %s", cmd.Path)
		h.T.Fatal(err)
		return types.WaitStatus{}, err
	}
	var cmdIdx int
	for i, arg := range cmd.Args {
		if arg == "--" {
			cmdIdx = i + 1
			break
		}
	}
	if cmdIdx == 0 {
		err := fmt.Errorf("missing expected sudo argument '--': %s", cmd.Args)
		h.T.Fatal(err)
		return types.WaitStatus{}, err
	}
	cmd.Path = cmd.Args[cmdIdx]
	cmd.Args = cmd.Args[cmdIdx+1:]
	return h.BaseHost.Run(ctx, cmd)
}

func (h HostLocalRunSudoOnlyTest) String() string {
	return h.BaseHost.String()
}

func (h HostLocalRunSudoOnlyTest) Type() string {
	return h.BaseHost.Type()
}

func (h HostLocalRunSudoOnlyTest) Close(ctx context.Context) error {
	return h.BaseHost.Close(ctx)
}

func NewHostLocalRunSudoOnlyTest(t *testing.T) HostLocalRunSudoOnlyTest {
	localHost := Local{}
	host := HostLocalRunSudoOnlyTest{
		T:        t,
		BaseHost: localHost,
	}
	return host
}

func TestSudo(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	wrappedBaseHost := NewHostLocalRunSudoOnlyTest(t)

	baseHost, err := NewSudoWrapper(ctx, wrappedBaseHost)
	require.NoError(t, err)
	defer func() { require.NoError(t, baseHost.Close(ctx)) }()

	tempDirPrefix := t.TempDir()
	testBaseHost(t, ctx, tempDirPrefix, baseHost, wrappedBaseHost.String(), wrappedBaseHost.Type())
}
