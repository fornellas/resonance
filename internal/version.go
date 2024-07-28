package internal

import (
	_ "embed"
	"strings"
)

// Version is the version of the built binary
//
//go:generate sh -c "git describe --tags --always --dirty  > .version"
//go:embed .version
var Version string

func init() {
	Version = strings.TrimSuffix(Version, "\n")
}
