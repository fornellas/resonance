package internal

import (
	_ "embed"
	"os"
	"strings"
)

// Path to Git root directory for resonance sources.
//
//go:generate sh -c "git rev-parse --show-toplevel  > .git-toplevel"
//go:embed .git-toplevel
var GitTopLevel string

func init() {
	GitTopLevel = strings.TrimSuffix(GitTopLevel, "\n") + string(os.PathSeparator)
}
