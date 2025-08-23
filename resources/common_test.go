package resources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
)

// runAndRequireSuccess runs a command and requires it to succeed, providing detailed error info on failure
func runAndRequireSuccess(t *testing.T, ctx context.Context, host types.BaseHost, cmd types.Cmd) string {
	waitStatus, stdout, stderr, err := lib.Run(ctx, host, cmd)
	require.NoError(t, err)
	require.True(t, waitStatus.Success(), "Command %s %v failed: %s\nSTDOUT:\n%s\nSTDERR:\n%s", cmd.Path, cmd.Args, waitStatus.String(), stdout, stderr)
	return stdout
}
