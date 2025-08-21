package state

import (
	"github.com/fornellas/resonance/resources"
)

// State represents a desired host state, for all managed resources.
type State struct {
	Files       []*resources.File       `hcl:"file,block"`
	APTPackages []*resources.APTPackage `hcl:"apt_package,block"`
}
