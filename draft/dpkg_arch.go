package draft

import (
	"context"
	"sort"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/resources"
)

// DpkgArch manages the set of foreign architectures that dpkg is configured to support.
// This allows installing packages built for architectures other than the system's native one,
// enabling multiarch support as described in https://wiki.debian.org/Multiarch/HOWTO.
//
// The ForeignArchitectures field lists all extra architectures to be enabled.
type DpkgArch struct {
	// ForeignArchitectures specifies extra architectures dpkg is configured to allow packages to be
	// installed for. Required.
	ForeignArchitectures []string
}

// Loads the full state of DpkgArch from given host.
func LoadDpkgArch(ctx context.Context, host types.Host) (*DpkgArch, error) {
	panic("TODO")
}

func (d *DpkgArch) ID() string {
	return "DpkgArch"
}

func (a *DpkgArch) Satisfies(ctx context.Context, host types.Host, otherResource resources.Resource) (bool, error) {
	panic("TODO")
}

func (a *DpkgArch) Validate() error {
	panic("TODO")
}

func (a *DpkgArch) Merge(otherResource resources.Resource) error {
	currentArchs := map[string]bool{}
	for _, currenTarch := range a.ForeignArchitectures {
		currentArchs[currenTarch] = true
	}
	for _, newArch := range otherResource.(*DpkgArch).ForeignArchitectures {
		if _, ok := currentArchs[newArch]; !ok {
			a.ForeignArchitectures = append(a.ForeignArchitectures, newArch)
		}
	}
	sort.Strings(a.ForeignArchitectures)
	return nil
}

func (a *DpkgArch) Apply(ctx context.Context, host types.Host) error {
	panic("TODO")
}
