package host

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/log"
)

func TestHttpAgent(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	host, err := NewHttpAgent(ctx, Local{})
	defer func() { require.NoError(t, host.Close(ctx)) }()
	require.NoError(t, err)
	testHost(t, host)
}
