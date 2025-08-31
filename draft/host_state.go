package draft

import (
	"context"
	"fmt"

	"github.com/fornellas/resonance/host/types"
)

// HostState holds the state of all managed resources for a host.
// TODO Consider making HostState generic:
//
//	type HostState struct {
//	    Resources map[string]Resource // Key is Resource type + Resource.ID()
//	}
type HostState struct {
	APTPackages APTPackages
	DpkgArch    *DpkgArch
	Files       Files
}

// Add given resource to HostState. Caller must ensure that no two resources with same ID are
// added.
func (hs *HostState) AddResource(resource Resource) {
	switch r := resource.(type) {
	case *APTPackage:
		hs.APTPackages = append(hs.APTPackages, r)
	case *DpkgArch:
		hs.DpkgArch = r
	case *File:
		hs.Files = append(hs.Files, r)
	default:
		panic(fmt.Sprintf("bug: unknown resource type: %T", resource))
	}
}

// Gets a Resource with same ID().
func (hs *HostState) GetResource(resource Resource) (Resource, bool) {
	switch r := resource.(type) {
	case *APTPackage:
		for _, aptPackage := range hs.APTPackages {
			if r.ID() == aptPackage.ID() {
				return aptPackage, true
			}
		}
	case *DpkgArch:
		if hs.DpkgArch != nil {
			return hs.DpkgArch, true
		}
	case *File:
		for _, file := range hs.Files {
			if r.ID() == file.ID() {
				return file, true
			}
		}
	default:
		panic(fmt.Sprintf("bug: unknown resource type: %T", resource))
	}
	return nil, false
}

// Get a list of all resources.
func (hs *HostState) GetResources() []Resource {
	resources := []Resource{}
	for _, aptPackage := range hs.APTPackages {
		resources = append(resources, aptPackage)
	}
	if hs.DpkgArch != nil {
		resources = append(resources, hs.DpkgArch)
	}
	for _, file := range hs.Files {
		resources = append(resources, file)
	}
	return resources
}

// Applies the state of all resources to host.
func (hs *HostState) Apply(ctx context.Context, host types.Host) error {
	panic("TODO")
}

// Load the full host state, for all resources.
func (hs *HostState) Load(ctx context.Context, host types.Host) (*HostState, error) {
	panic("TODO")
}

// Satisfies return true when self satisfies the state required by other.
func (hs *HostState) Satisfies(ctx context.Context, host types.Host, otherHostState *HostState) (bool, error) {
	for _, otherResource := range otherHostState.GetResources() {
		resource, ok := hs.GetResource(otherResource)
		if !ok {
			return false, nil
		}
		ok, err := resource.Satisfies(ctx, host, otherResource)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}
