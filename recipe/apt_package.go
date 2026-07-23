package recipe

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

// APTPackage manages APT packages.
type APTPackage struct {
	// The name of the package.
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#package
	Package string `yaml:"package" json:"package" jsonschema:"required,description=The name of the package. See https://www.debian.org/doc/debian-policy/ch-controlfields.html#package"`

	// Whether to remove the package.
	Absent bool `yaml:"absent" json:"absent,omitempty" jsonschema:"description=Whether to remove the package."`

	// Architectures.
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#architecture
	Architectures []string `yaml:"architectures" json:"architectures,omitempty" jsonschema:"description=Architectures. See https://www.debian.org/doc/debian-policy/ch-controlfields.html#architecture"`

	// Package version.
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	Version string `yaml:"version" json:"version,omitempty" jsonschema:"description=Package version. See https://www.debian.org/doc/debian-policy/ch-controlfields.html#version"`

	// Location is where this APTPackage was declared.
	Location Location `yaml:"-" json:"-"`
}

// UnmarshalYAML implements yaml.NodeUnmarshalerContext: it decodes
// APTPackage as usual, additionally capturing where it was declared, in a
// single pass — no separate lookup against the source is needed afterwards.
func (p *APTPackage) UnmarshalYAML(ctx context.Context, node ast.Node) error {
	type PlainAPTPackage APTPackage
	var plainAPTPackage PlainAPTPackage
	if err := yaml.NodeToValue(node, &plainAPTPackage); err != nil {
		return err
	}
	*p = APTPackage(plainAPTPackage)
	p.Location = locationFromNode(ctx, node)

	return nil
}

// Validate checks that the APTPackage is well formed.
func (p APTPackage) Validate() error {
	if p.Package == "" {
		return fmt.Errorf("package is required")
	}
	if !isValidPackageName(p.Package) {
		return fmt.Errorf("package %q is not a valid package name", p.Package)
	}
	return nil
}

// isValidPackageName reports whether name is a valid Debian package name: at
// least two characters, starting with a lowercase letter or digit, and
// containing only lowercase letters, digits, '+', '-' and '.'.
// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#package
func isValidPackageName(name string) bool {
	if len(name) < 2 {
		return false
	}
	for i, r := range name {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if i == 0 {
			if !isAlphaNum {
				return false
			}
			continue
		}
		if !isAlphaNum && r != '+' && r != '-' && r != '.' {
			return false
		}
	}
	return true
}
