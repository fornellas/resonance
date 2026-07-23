package main

import (
	"testing"
)

func TestValidateCmd(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		tc := TestCmd{
			Args:                 []string{"validate", "testdata/valid.yaml"},
			ExpectedCode:         0,
			ExpectStderrContains: []string{"Recipe is valid"},
		}
		tc.Run(t)
	})

	t.Run("valid directory", func(t *testing.T) {
		tc := TestCmd{
			Args:                 []string{"validate", "testdata/dir"},
			ExpectedCode:         0,
			ExpectStderrContains: []string{"Recipe is valid"},
		}
		tc.Run(t)
	})

	t.Run("multiple args merged", func(t *testing.T) {
		tc := TestCmd{
			Args:                 []string{"validate", "testdata/valid.yaml", "testdata/dir"},
			ExpectedCode:         0,
			ExpectStderrContains: []string{"Recipe is valid"},
		}
		tc.Run(t)
	})

	t.Run("invalid recipe", func(t *testing.T) {
		tc := TestCmd{
			Args:         []string{"validate", "testdata/invalid.yaml"},
			ExpectedCode: 1,
			ExpectStderrContains: []string{
				`testdata/invalid.yaml:2:9: mode "644" must be in octal notation (eg: 0644)`,
			},
		}
		tc.Run(t)
	})

	t.Run("resource declared in two different files", func(t *testing.T) {
		tc := TestCmd{
			Args:         []string{"validate", "testdata/dup-a.yaml", "testdata/dup-b.yaml"},
			ExpectedCode: 1,
			ExpectStderrContains: []string{
				`testdata/dup-b.yaml:2:9: file "/etc/dup" already declared at testdata/dup-a.yaml:2:9`,
			},
		}
		tc.Run(t)
	})

	t.Run("missing path", func(t *testing.T) {
		tc := TestCmd{
			Args:         []string{"validate", "testdata/does-not-exist.yaml"},
			ExpectedCode: 1,
			ExpectStderrContains: []string{
				"stat testdata/does-not-exist.yaml: no such file or directory",
			},
		}
		tc.Run(t)
	})
}
