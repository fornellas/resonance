package resource

import "github.com/fornellas/resonance/resources"

// DiffResourceState diffs two resource states, by using DiffableManageableResource if the
// resource implements this interface, otherwise, use DiffAsYaml.
func DiffResourceState(manageableResource resources.ManageableResource, a, b resources.State) resources.Chunks {
	diffableManageableResource, ok := manageableResource.(resources.DiffableManageableResource)
	if ok {
		return diffableManageableResource.Diff(a, b)
	} else {
		return resources.DiffAsYaml(a, b)
	}
}
