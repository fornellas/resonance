package host

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/slogxt/log"
)

func TestAgentClientWrapper(t *testing.T) {
	ctx := t.Context()
	ctx = log.WithTestLogger(ctx)

	baseHost := Local{}
	host, err := NewAgentClientWrapper(ctx, baseHost)
	require.NoError(t, err)
	defer func() { require.NoError(t, host.Close(ctx)) }()

	testHost(t, ctx, host, baseHost.String(), baseHost.Type())
}
