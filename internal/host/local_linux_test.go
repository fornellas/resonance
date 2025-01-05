package host

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocal(t *testing.T) {
	host := Local{}
	ctx := context.Background()
	defer func() { require.NoError(t, host.Close(ctx)) }()
	testHost(t, host)
}
