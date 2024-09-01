package internal

import (
	_ "embed"
	"fmt"
	"runtime"
	"strings"
)

// Version is the version of the built binary
//
//go:generate sh -c "git describe --tags --always --dirty  > .version"
//go:embed .version
var Version string

func init() {
	Version = fmt.Sprintf(
		"%s.%s.%s",
		strings.TrimSuffix(Version, "\n"),
		runtime.GOOS, runtime.GOARCH,
	)
}
