//go:build !skip_integration

package host

import (
	"testing"
)

func TestDockerIntegration(t *testing.T) {
	dockerHost, connection := GetTestDockerHost(t, "debian")

	testBaseHost(t, dockerHost, connection, "docker")
}
