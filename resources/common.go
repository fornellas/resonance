package resources

import "regexp"

var validDpkgArchitectureRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]+$`)

var testDockerImages = []string{
	"debian:bookworm",
	"debian:trixie",
	"ubuntu:22.04",
	"ubuntu:24.04",
}
