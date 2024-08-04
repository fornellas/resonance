//go:build tools
// +build tools

package main

import (
	_ "github.com/client9/misspell/cmd/misspell"
	_ "github.com/fornellas/rrb"
	_ "github.com/fzipp/gocyclo/cmd/gocyclo"
	_ "github.com/jandelgado/gcov2lcov"
	_ "github.com/rakyll/gotest"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/vuln/cmd/govulncheck"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
