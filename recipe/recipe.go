// Package recipe defines the schema for declaring the desired state of a host.
package recipe

//go:generate go run ./internal/genschema

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

// Recipe declares the desired state of a host.
type Recipe struct {
	Files       []File       `yaml:"files" json:"files,omitempty" jsonschema:"description=Files to be managed on the host."`
	APTPackages []APTPackage `yaml:"apt_packages" json:"apt_packages,omitempty" jsonschema:"description=APT packages to be managed on the host."`
}

// Validate checks that every resource in the Recipe is well formed. It does
// not check for resources declared more than once — see Merge for that.
func (r Recipe) Validate() error {
	for _, file := range r.Files {
		if err := file.Validate(); err != nil {
			return &ValidationError{Location: file.Location, Err: err}
		}
	}
	for _, pkg := range r.APTPackages {
		if err := pkg.Validate(); err != nil {
			return &ValidationError{Location: pkg.Location, Err: err}
		}
	}
	return nil
}

// Merge appends other's resources into r, returning a
// *DuplicateResourceError if any resource in other is already declared in
// r.
func (r *Recipe) Merge(other *Recipe) error {
	declaredFileAt := map[string]Location{}
	for _, f := range r.Files {
		declaredFileAt[f.Path] = f.Location
	}
	if err := detectDuplicates(
		"file",
		other.Files,
		func(f File) string { return f.Path },
		func(f File) Location { return f.Location },
		declaredFileAt,
	); err != nil {
		return err
	}

	declaredPackageAt := map[string]Location{}
	for _, p := range r.APTPackages {
		declaredPackageAt[p.Package] = p.Location
	}
	if err := detectDuplicates(
		"apt_package",
		other.APTPackages,
		func(p APTPackage) string { return p.Package },
		func(p APTPackage) Location { return p.Location },
		declaredPackageAt,
	); err != nil {
		return err
	}

	r.Files = append(r.Files, other.Files...)
	r.APTPackages = append(r.APTPackages, other.APTPackages...)

	return nil
}

// Parse reads a Recipe from its YAML representation, validating it. path
// identifies the source (eg: a file path), and is used to locate any
// reported error, as well as each declared resource's own Location.
func Parse(ctx context.Context, yamlBytes []byte, path string) (*Recipe, error) {
	ctx = context.WithValue(ctx, pathContextKey{}, path)

	var r Recipe
	if err := yaml.UnmarshalContext(ctx, yamlBytes, &r); err != nil {
		return nil, fmt.Errorf("%s: failed to parse recipe: %w", path, err)
	}

	if err := r.Validate(); err != nil {
		return nil, err
	}

	// Merging into an empty Recipe reuses Merge's own duplicate detection to
	// catch resources declared more than once within r itself.
	merged := &Recipe{}
	if err := merged.Merge(&r); err != nil {
		return nil, err
	}

	return merged, nil
}

// pathContextKey is the context.Context key under which the source path
// passed to Parse is stored, so UnmarshalYAML can read it back while decoding.
type pathContextKey struct{}

// locationFromNode returns the Location that node was declared at, taking
// the source path from ctx (see pathFromContext). It's the one part of a
// resource's UnmarshalYAML that never depends on the resource's own type.
func locationFromNode(ctx context.Context, node ast.Node) Location {
	pos := node.GetToken().Position
	path, ok := ctx.Value(pathContextKey{}).(string)
	if !ok {
		panic("bug: context missing pathContextKey")
	}
	return Location{Path: path, Line: pos.Line, Column: pos.Column}
}
