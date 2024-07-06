package internal

import (
	_ "embed"
	"os"
	"strings"
)

//go:generate sh -c "git rev-parse --show-toplevel  > .git-toplevel"
//go:embed .git-toplevel
var GitTopLevel string

func init() {
	GitTopLevel = strings.TrimSuffix(GitTopLevel, "\n") + string(os.PathSeparator)
}
