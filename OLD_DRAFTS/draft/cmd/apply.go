package cmd

import (
	"context"

	hostPkg "github.com/fornellas/resonance/draft/host"
	resourcePkg "github.com/fornellas/resonance/draft/resource"
	storePkg "github.com/fornellas/resonance/draft/store"
)

func saveOriginal(
	ctx context.Context,
	store storePkg.Store,
	host hostPkg.Host,
	states resourcePkg.States,
) error {
	// Save Original
	idsToSaveOriginal := []*resourcePkg.Id{}
	for _, state := range states {
		hasOriginal, err := store.HasOriginalId(ctx, state.Id())
		if err != nil {
			return err
		}
		if hasOriginal {
			continue
		}
		idsToSaveOriginal = append(idsToSaveOriginal, state.Id())
	}
	originalStatesToSave, err := resourcePkg.Load(ctx, host, idsToSaveOriginal)
	if err != nil {
		return err
	}
	for _, state := range originalStatesToSave {
		if err := store.SaveOriginal(ctx, state); err != nil {
			return err
		}
	}
	return nil
}

func stage(
	ctx context.Context,
	store storePkg.Store,
	states resourcePkg.States,
) (resourcePkg.States, resourcePkg.States, error) {
	previousStates := resourcePkg.States{}

	// TODO set previousStates to committed if exists or original if not

	committedIds, err := store.ListCommittedIds(ctx)
	if err != nil {
		return nil, nil, err
	}
	restoreStates := resourcePkg.States{}
	for _, id := range committedIds {
		if states.HasId(id) {
			continue
		}
		state, err := store.GetOriginal(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		restoreStates = append(restoreStates, state)
	}
	stageStates := append(states, restoreStates...)
	if err := store.Stage(ctx, stageStates); err != nil {
		return nil, nil, err
	}
	return stageStates, previousStates, nil
}

func Apply(
	ctx context.Context,
	path string,
	store storePkg.Store,
	host hostPkg.Host,
) error {
	ok, err := store.HasStaged(ctx)
	if err != nil {
		return err
	}
	if ok {
		// TODO retry
		// TODO abort
		//   If committed
		//     stage last commit
		//     apply
		//   else
		//     apply all original (order? sholud we commit all original?)
		panic("TODO")
	}

	states, err := resourcePkg.LoadPath(ctx, path)
	if err != nil {
		return err
	}

	// TODO if committed, check state

	if err := saveOriginal(ctx, store, host, states); err != nil {
		return err
	}

	// Stage
	var stagedStates resourcePkg.States
	var previousStates resourcePkg.States
	if stagedStates, previousStates, err = stage(ctx, store, states); err != nil {
		return err
	}

	// TODO diff previousStates to stageStates & ask for confirmation

	// Apply
	if err := resourcePkg.Apply(ctx, host, stagedStates); err != nil {
		return err
	}

	// TODO validate: load state & compare to staged

	// Commit
	// FIXME we stage both states and restoreStates, but only states should be part of the commit:
	// restoreStates were restored to original, and can be deleted from original as well
	if err := store.Commit(ctx); err != nil {
		return nil
	}

	// Delete Original if not Committed
	originalIds, err := store.ListOriginalIds(ctx)
	if err != nil {
		return err
	}
	for _, id := range originalIds {
		if stagedStates.HasId(id) {
			continue
		}
		if err := store.DeleteOriginal(ctx, id); err != nil {
			return nil
		}
	}

	return nil
}
