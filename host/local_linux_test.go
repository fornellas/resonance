package host

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/log"
)

func TestLocal(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	host := Local{}
	defer func() { require.NoError(t, host.Close(ctx)) }()

	tempDirPrefix := t.TempDir()
	testHost(t, ctx, tempDirPrefix, host, "localhost", "localhost")
}
