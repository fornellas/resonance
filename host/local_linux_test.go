package host

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocal(t *testing.T) {
	host := Local{}
	defer func() { require.NoError(t, host.Close()) }()
	testHost(t, host)
}
