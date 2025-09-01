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
		originalHostState.MustAppendResource(storedOriginalResource)
	}

	toLoadHostState := &HostState{}

	updatedOriginalHostState := false

	for _, targetResource := range targetHostState.GetResources() {
		if _, ok := originalHostState.GetResourceByID(targetResource); ok {
			continue
		}
		updatedOriginalHostState = true
		toLoadHostState.MustAppendResource(targetResource)
	}

	if updatedOriginalHostState {
		extraOriginalHostState, err := toLoadHostState.Load(ctx, host)
		if err != nil {
			return nil, err
		}

		for _, extraOriginalResource := range extraOriginalHostState.GetResources() {
			originalHostState.MustAppendResource(extraOriginalResource)
		}

		if err := store.SaveOriginalHostState(ctx, originalHostState); err != nil {
			return nil, err
		}
	}

	return originalHostState, nil
}

func preparePlannedHostState(ctx context.Context, store Store, originalHostState *HostState, targetHostState *HostState) (*HostState, error) {
	plannedHostState := &HostState{}
	targetResources := targetHostState.GetResources()
	for _, targetResource := range targetResources {
		plannedHostState.MustAppendResource(targetResource)
	}
	for _, originalResource := range originalHostState.GetResources() {
		if _, ok := targetHostState.GetResourceByID(originalResource); !ok {
			plannedHostState.MustAppendResource(originalResource)
		}
	}

	if err := store.SavePlannedHostState(ctx, plannedHostState); err != nil {
		return nil, err
	}

	return plannedHostState, nil
}

func cleanupOriginalHostState(ctx context.Context, store Store, originalHostState *HostState, targetHostState *HostState) error {
	cleanedOriginalHostState := &HostState{}
	updated := false
	for _, originalResource := range originalHostState.GetResources() {
		if _, ok := targetHostState.GetResourceByID(originalResource); ok {
			cleanedOriginalHostState.MustAppendResource(originalResource)
			updated = true
		}
	}
	if updated {
		return store.SaveOriginalHostState(ctx, cleanedOriginalHostState)
	}
	return nil
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

	if err := cleanupOriginalHostState(ctx, store, originalHostState, targetHostState); err != nil {
		return err
	}

	return nil
}
