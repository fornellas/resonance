package host

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

type localRunSudoOnly struct {
	baseRunOnly
	T    *testing.T
	Host Host
}

func (lrso localRunSudoOnly) Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
	if cmd.Path != "sudo" {
		err := fmt.Errorf("attempted to run non-sudo command: %s", cmd.Path)
		lrso.T.Fatal(err)
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
		lrso.T.Fatal(err)
		return types.WaitStatus{}, err
	}
	cmd.Path = cmd.Args[cmdIdx]
	cmd.Args = cmd.Args[cmdIdx+1:]
	return lrso.Host.Run(ctx, cmd)
}

func newLocalRunSudoOnly(t *testing.T, host Host) localRunSudoOnly {
	run := localRunSudoOnly{
		T:    t,
		Host: host,
	}
	run.baseRunOnly.T = t
	run.baseRunOnly.Host = host
	return run
}

func TestSudo(t *testing.T) {
	ctx := context.Background()
	ctx = log.SetLoggerValue(ctx, os.Stderr, "info", func(code int) {
		t.Fatalf("exit called with %d", code)
	})

	host, err := NewSudo(ctx, newLocalRunSudoOnly(t, Local{}))
	require.NoError(t, err)
	defer func() { require.NoError(t, host.Close()) }()
	testHost(t, host)
}
