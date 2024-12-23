//go:build tools
// +build tools

package main

import (
	_ "github.com/client9/misspell/cmd/misspell"
	_ "github.com/fornellas/rrb"
	_ "github.com/fzipp/gocyclo/cmd/gocyclo"
	_ "github.com/gordonklaus/ineffassign"
	_ "github.com/jandelgado/gcov2lcov"
	_ "github.com/rakyll/gotest"
	_ "github.com/yoheimuta/protolint/cmd/protolint"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/vuln/cmd/govulncheck"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
