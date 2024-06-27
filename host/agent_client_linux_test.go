package host

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/log"
)

func TestAgent(t *testing.T) {
	ctx := context.Background()
	ctx = log.SetLoggerValue(ctx, os.Stderr, "info", func(code int) {
		t.Fatalf("exit called with %d", code)
	})

	host, err := NewAgent(ctx, Local{})
	defer func() { require.NoError(t, host.Close()) }()
	require.NoError(t, err)
	testHost(t, host)
}
