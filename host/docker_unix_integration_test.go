//go:build !skip_integration

package host

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDockerIntegration(t *testing.T) {
	dockerHost, connection := GetTestDockerHost(t, "debian")

	defer func() { require.NoError(t, dockerHost.Close(t.Context())) }()

	testBaseHost(t, dockerHost, connection, "docker")
}
