package draft

import (
	"context"
	"fmt"

	"github.com/fornellas/resonance/host/types"
)

func checkStoredPlannedHostState(ctx context.Context, store Store) error {
	plannedHostState, err := store.GetPlannedHostState(ctx)
	if err != nil {
		return err
	}
	if plannedHostState != nil {
		return fmt.Errorf("previous apply interrupted")
	}
	return nil
}

func checkCommittedHostState(ctx context.Context, host types.Host, store Store) error {
	committedHostState, err := store.GetCommittedHostState(ctx)
	if err != nil {
		return err
	}
	if committedHostState != nil {
		currentHostState, err := committedHostState.Load(ctx, host)
		if err != nil {
			return err
		}
		ok, err := currentHostState.Satisfies(ctx, host, committedHostState)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("committed host state changed")
		}
	}
	return nil
}

func prepareOriginalHostState(ctx context.Context, host types.Host, store Store, targetHostState *HostState) (*HostState, error) {
	storedOriginalHostState, err := store.GetOriginalHostState(ctx)
	if err != nil {
		return nil, err
	}

	originalHostState := &HostState{}

	for _, storedOriginalResource := range storedOriginalHostState.GetResources() {
		originalHostState.AddResource(storedOriginalResource)
	}

	toLoadHostState := &HostState{}

	updatedOriginalHostState := false

	for _, targetResource := range targetHostState.GetResources() {
		if _, ok := originalHostState.GetResource(targetResource); ok {
			continue
		}
		updatedOriginalHostState = true
		toLoadHostState.AddResource(targetResource)
	}

	if updatedOriginalHostState {
		extraOriginalHostState, err := toLoadHostState.Load(ctx, host)
		if err != nil {
			return nil, err
		}

		for _, extraOriginalResource := range extraOriginalHostState.GetResources() {
			originalHostState.AddResource(extraOriginalResource)
		}

		if err := store.SaveOriginalHostState(ctx, originalHostState); err != nil {
			return nil, err
		}
	}

	return originalHostState, nil
}

func planResources[R Resource](targetResources []R, originalResources []R) []R {
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

func preparePlannedHostState(ctx context.Context, store Store, originalHostState *HostState, targetHostState *HostState) (*HostState, error) {
	plannedHostState := &HostState{}

	plannedHostState.APTPackages = planResources(targetHostState.APTPackages, originalHostState.APTPackages)

	if targetHostState.DpkgArch != nil {
		plannedHostState.DpkgArch = targetHostState.DpkgArch
	} else if originalHostState.DpkgArch != nil {
		plannedHostState.DpkgArch = originalHostState.DpkgArch
	}

	plannedHostState.Files = planResources(targetHostState.Files, originalHostState.Files)

	if err := store.SavePlannedHostState(ctx, plannedHostState); err != nil {
		return nil, err
	}

	return plannedHostState, nil
}

func Apply(ctx context.Context, host types.Host, store Store, targetHostState *HostState) error {
	if err := checkStoredPlannedHostState(ctx, store); err != nil {
		return err
	}

	if err := checkCommittedHostState(ctx, host, store); err != nil {
		return err
	}

	originalHostState, err := prepareOriginalHostState(ctx, host, store, targetHostState)
	if err != nil {
		return err
	}

	plannedHostState, err := preparePlannedHostState(ctx, store, originalHostState, targetHostState)
	if err != nil {
		return err
	}

	if err := plannedHostState.Apply(ctx, host); err != nil {
		return err
	}

	if err := store.CommitPlannedHostState(ctx); err != nil {
		return err
	}

	// TODO update original to only have committed state

	return nil
}
