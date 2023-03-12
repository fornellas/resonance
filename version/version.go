package version

import (
	_ "embed"
	"strings"
)

// Version is the version of the built binary
type Version string

//go:generate sh -c "git describe --tags --always  > .version"
//go:embed .version
var version string

// IsCurrent returns whether Version is the same as the current binary.
func (v Version) IsCurrent() bool {
	return v == GetVersion()
}

// GetVersion returns the Version for the current binary.
func GetVersion() Version {
	return Version(strings.TrimSuffix(version, "\n"))
}
