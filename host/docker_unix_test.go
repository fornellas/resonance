package host

import (
	"testing"
)

func TestDocker(t *testing.T) {
	dockerHost, connection := GetTestDockerHost(t, "debian")

	testBaseHost(t, dockerHost, connection, "docker")
}
