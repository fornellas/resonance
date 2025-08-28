package draft

import "context"

// planResources merges managed and restore resources using their ID() string.
func planResources[R interface{ ID() string }](targetResources []R, originalResources []R) []R {
	planResources := []R{}
	targetResourceIdMap := make(map[string]struct{}, len(targetResources))
	for _, targetResource := range targetResources {
		planResources = append(planResources, targetResource)
		targetResourceIdMap[targetResource.ID()] = struct{}{}
	}
	for _, r := range originalResources {
		if _, ok := targetResourceIdMap[r.ID()]; !ok {
			planResources = append(planResources, r)
		}
	}
	return planResources
}

// Plan changes to a host.
// originalState holds the original state of the resource. If it was previously applied, then it
// holds the state before the first apply. If not previously applied, it holds the current state
// of the resource at the host. This is used to restore original state of resources that stop being
// managed and to allow rollback on failed apply. States here are "full", as returned by
// Resource.Load.
// currentState holds the current state of the resource at the host, for all originalState and
// targetState resources. States here are "full", as returned by Resource.Load.
// targetState holds the state of resources that we want to manage when applying, which may be a
// different set of resources form originalState and currentState. Resources that are present here
// will be applied, resources that are at originalState but not at targetState will be restored
// to the original state. States here may be partial (eg: File with user name, missing UID), and
// final values will be calculated during apply.
// It returns a State that can be applied to a Host, to move it form currentState to targetState.
func Plan(ctx context.Context, originalState State, currentState State, targetState State) State {
	plannedState := State{}

	plannedState.APTPackages = planResources(targetState.APTPackages, originalState.APTPackages)

	if targetState.DpkgArch != nil {
		plannedState.DpkgArch = targetState.DpkgArch
	} else if originalState.DpkgArch != nil {
		plannedState.DpkgArch = originalState.DpkgArch
	}

	plannedState.Files = planResources(targetState.Files, originalState.Files)

	return plannedState
}
