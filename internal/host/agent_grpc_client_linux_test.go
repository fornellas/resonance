package host

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/log"
)

func TestGrpcAgent(t *testing.T) {
	t.Skip("Skipping this test for now")
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	host, err := NewGrpcAgent(ctx, Local{})
	defer func() { require.NoError(t, host.Close()) }()
	require.NoError(t, err)
	testHost(t, host)
}
