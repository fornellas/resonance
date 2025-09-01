package draft

import (
	"context"
	_ "embed"

	"github.com/fornellas/resonance/host/types"
)

//go:embed debconf_editor.sh
var debconfEditor string

// A debconf question.
// See https://wiki.debian.org/debconf
type DebconfQuestion string

// Debconf selections for a DebconfQuestion.
// See https://wiki.debian.org/debconf
type DebconfAnswer string

// APTPackage manages APT packages.
type APTPackage struct {
	// The name of the package. Must be set.
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#package
	Package string
	// Whether to remove the package. When true, other fields can't be set (and vice versa).
	Absent bool
	// Architectures. Optional. If empty, it defaults to dpkg --print-architecture value. If set,
	// install packages for all these Architectures.
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#architecture
	Architectures []string
	// Package version. Optional. If set, then Hold must be set to true (so the version is kept on
	// updates).
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	Version string
	// Whether the package should be held to prevent automatic upgrades. If set, also requires
	// Version to be set.
	Hold bool
	// Package debconf selections. Optional. It only affects values defined here, other values are
	// unchanged.
	// See https://wiki.debian.org/debconf
	DebconfSelections map[DebconfQuestion]DebconfAnswer
}

func (a *APTPackage) ID() string {
	return a.Package
}

func (a *APTPackage) Satisfies(ctx context.Context, host types.Host, otherResource Resource) (bool, error) {
	panic("TODO")
}

func (a *APTPackage) Validate() error {
	panic("TODO")
}

func (a *APTPackage) Merge(otherResource Resource) error {
	panic("TODO")
}

type APTPackages []*APTPackage

// Loads the full state of APTPackages from host for all given package names (IDs).
func LoadAPTPackages(ctx context.Context, host types.Host, names ...string) (APTPackages, error) {
	panic("TODO")
}

func (ap *APTPackages) Apply(ctx context.Context, host types.Host) error {
	panic("TODO")
}
