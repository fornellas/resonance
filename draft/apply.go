package draft

import (
	"context"
	"fmt"

	"github.com/fornellas/resonance/host/types"
	statePkg "github.com/fornellas/resonance/state"
	storePkg "github.com/fornellas/resonance/store"
)

func checkStoredPlannedState(ctx context.Context, store storePkg.Store) error {
	plannedState, err := store.GetPlannedState(ctx)
	if err != nil {
		return err
	}
	if plannedState != nil {
		return fmt.Errorf("previous apply interrupted")
	}
	return nil
}

func checkCommittedState(ctx context.Context, host types.Host, store storePkg.Store) error {
	committedState, err := store.GetCommittedState(ctx)
	if err != nil {
		return err
	}
	if committedState != nil {
		currentState, err := committedState.Load(ctx, host)
		if err != nil {
			return err
		}
		ok, err := currentState.Satisfies(ctx, host, committedState)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("committed host state changed")
		}
	}
	return nil
}

func prepareOriginalState(ctx context.Context, host types.Host, store storePkg.Store, targetState *statePkg.State) (*statePkg.State, error) {
	storedOriginalState, err := store.GetOriginalState(ctx)
	if err != nil {
		return nil, err
	}

	originalState := &statePkg.State{}

	for _, storedOriginalResource := range storedOriginalState.GetResources() {
		originalState.MustAppendResource(storedOriginalResource)
	}

	toLoadState := &statePkg.State{}

	updatedOriginalState := false

	for _, targetResource := range targetState.GetResources() {
		if _, ok := originalState.GetResourceByID(targetResource); ok {
			continue
		}
		updatedOriginalState = true
		toLoadState.MustAppendResource(targetResource)
	}

	if updatedOriginalState {
		extraOriginalState, err := toLoadState.Load(ctx, host)
		if err != nil {
			return nil, err
		}

		for _, extraOriginalResource := range extraOriginalState.GetResources() {
			originalState.MustAppendResource(extraOriginalResource)
		}

		if err := store.SaveOriginalState(ctx, originalState); err != nil {
			return nil, err
		}
	}

	return originalState, nil
}

func preparePlannedState(ctx context.Context, store storePkg.Store, originalState *statePkg.State, targetState *statePkg.State) (*statePkg.State, error) {
	plannedState := &statePkg.State{}
	targetResources := targetState.GetResources()
	for _, targetResource := range targetResources {
		plannedState.MustAppendResource(targetResource)
	}
	for _, originalResource := range originalState.GetResources() {
		if _, ok := targetState.GetResourceByID(originalResource); !ok {
			plannedState.MustAppendResource(originalResource)
		}
	}

	if err := store.SavePlannedState(ctx, plannedState); err != nil {
		return nil, err
	}

	return plannedState, nil
}

func cleanupOriginalState(ctx context.Context, store storePkg.Store, originalState *statePkg.State, targetState *statePkg.State) error {
	cleanedOriginalState := &statePkg.State{}
	updated := false
	for _, originalResource := range originalState.GetResources() {
		if _, ok := targetState.GetResourceByID(originalResource); ok {
			cleanedOriginalState.MustAppendResource(originalResource)
			updated = true
		}
	}
	if updated {
		return store.SaveOriginalState(ctx, cleanedOriginalState)
	}
	return nil
}

func Apply(ctx context.Context, host types.Host, store storePkg.Store, targetState *statePkg.State) error {
	if err := checkStoredPlannedState(ctx, store); err != nil {
		return err
	}

	if err := checkCommittedState(ctx, host, store); err != nil {
		return err
	}

	originalState, err := prepareOriginalState(ctx, host, store, targetState)
	if err != nil {
		return err
	}

	plannedState, err := preparePlannedState(ctx, store, originalState, targetState)
	if err != nil {
		return err
	}

	if err := plannedState.Apply(ctx, host); err != nil {
		return err
	}

	if err := store.CommitPlannedState(ctx); err != nil {
		return err
	}

	if err := cleanupOriginalState(ctx, store, originalState, targetState); err != nil {
		return err
	}

	return nil
}
