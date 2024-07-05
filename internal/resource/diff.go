package resource

import "github.com/fornellas/resonance/resources"

// DiffResourceState diffs two resource states, by using DiffableResource if the
// resource implements this interface, otherwise, use DiffAsYaml.
func DiffResourceState(resource resources.Resource, a, b resources.State) resources.Chunks {
	diffableResource, ok := resource.(resources.DiffableResource)
	if ok {
		return diffableResource.Diff(a, b)
	} else {
		return resources.DiffAsYaml(a, b)
	}
}
