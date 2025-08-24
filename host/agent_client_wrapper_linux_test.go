package host

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/slogxt/log"
)

func TestAgentClientWrapper(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	baseHost := Local{}
	host, err := NewAgentClientWrapper(ctx, baseHost)
	defer func() { require.NoError(t, host.Close(ctx)) }()
	require.NoError(t, err)

	testHost(t, ctx, host, baseHost.String(), baseHost.Type())
}
