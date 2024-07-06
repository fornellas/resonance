package host

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/log"
)

func TestAgent(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	host, err := NewAgent(ctx, Local{})
	defer func() { require.NoError(t, host.Close()) }()
	require.NoError(t, err)
	testHost(t, host)
}
