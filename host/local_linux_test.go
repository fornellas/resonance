package host

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/slogxt/log"
)

func TestLocal(t *testing.T) {
	ctx := t.Context()
	ctx = log.WithTestLogger(ctx)

	host := Local{}
	defer func() { require.NoError(t, host.Close(ctx)) }()

	testHost(t, ctx, host, "localhost", "localhost")
}
