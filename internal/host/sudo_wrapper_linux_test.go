package host

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

type HostLocalRunSudoOnlyTest struct {
	BaseHostNoCallsTest
	T    *testing.T
	Host host.Host
}

func (h HostLocalRunSudoOnlyTest) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	if cmd.Path != "sudo" {
		err := fmt.Errorf("attempted to run non-sudo command: %s", cmd.Path)
		h.T.Fatal(err)
		return host.WaitStatus{}, err
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
		return host.WaitStatus{}, err
	}
	cmd.Path = cmd.Args[cmdIdx]
	cmd.Args = cmd.Args[cmdIdx+1:]
	return h.Host.Run(ctx, cmd)
}

func NewHostLocalRunSudoOnlyTest(t *testing.T) HostLocalRunSudoOnlyTest {
	localHost := Local{}
	host := HostLocalRunSudoOnlyTest{
		T:    t,
		Host: localHost,
	}
	host.BaseHostNoCallsTest.T = t
	host.BaseHostNoCallsTest.Host = localHost
	return host
}

func TestSudo(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	host, err := NewSudoWrapper(ctx, NewHostLocalRunSudoOnlyTest(t))
	require.NoError(t, err)
	defer func() { require.NoError(t, host.Close(ctx)) }()
	testHost(t, host)
}
