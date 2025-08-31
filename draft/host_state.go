package draft

import (
	"context"
	"reflect"

	"github.com/fornellas/resonance/host/types"
)

// HostState holds the state of all managed resources for a host.
type HostState struct {
	resources []Resource
}

// Add given resource to HostState. Caller must ensure that no two resources with same ID are
// added.
func (hs *HostState) AddResource(resource Resource) {
	hs.resources = append(hs.resources, resource)
}

// Gets a Resource with same ID().
func (hs *HostState) GetResourceByID(resource Resource) (Resource, bool) {
	for _, r := range hs.resources {
		if reflect.TypeOf(r) != reflect.TypeOf(resource) {
			continue
		}
		if r.ID() == resource.ID() {
			return r, true
		}
	}
	return nil, false
}

// Get a list of all resources.
func (hs *HostState) GetResources() []Resource {
	return hs.resources
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
		resource, ok := hs.GetResourceByID(otherResource)
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
