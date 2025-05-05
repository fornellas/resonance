package cmd

import (
	"context"

	hostPkg "github.com/fornellas/resonance/draft/host"
	resourcePkg "github.com/fornellas/resonance/draft/resource"
	storePkg "github.com/fornellas/resonance/draft/store"
)

//gocyclo:ignore
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
		// TODO abort
		//   If committed
		//     stage last commit
		//     apply
		//   else
		//     apply all original (order? sholud we commit all original?)
		// TODO retry
		panic("TODO")
	}

	states, err := resourcePkg.LoadPath(ctx, path)
	if err != nil {
		return err
	}

	// TODO if committed, check state

	// Save Original
	idsToSaveOriginal := []*resourcePkg.Id{}
	for _, state := range states {
		hasOriginal, err := store.HasOriginalId(ctx, state.Id())
		if err != nil {
			return err
		}
		if !hasOriginal {
			idsToSaveOriginal = append(idsToSaveOriginal, state.Id())
		}
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

	// Stage
	committedIds, err := store.ListCommittedIds(ctx)
	if err != nil {
		return err
	}
	restoreStates := resourcePkg.States{}
	for _, id := range committedIds {
		if states.HasId(id) {
			continue
		}
		state, err := store.GetOriginal(ctx, id)
		if err != nil {
			return err
		}
		restoreStates = append(restoreStates, state)
	}
	// FIXME merge restoreStates + states, respecting committedIds ordering
	stageStates := append(states, restoreStates...)
	if err := store.Stage(ctx, stageStates); err != nil {
		return err
	}

	// Apply
	// TODO calculate diff, from store (original/committed) to stageStates
	if err := resourcePkg.Apply(ctx, host, stageStates); err != nil {
		return err
	}

	// Commit
	if err := store.Commit(ctx); err != nil {
		return nil
	}

	// Delete Original if not Committed
	originalIds, err := store.ListOriginalIds(ctx)
	if err != nil {
		return err
	}
	for _, id := range originalIds {
		ok, err := store.CommitHas(ctx, id)
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		if err := store.DeleteOriginal(ctx, id); err != nil {
			return nil
		}
	}

	// TODO validate: load state & compare to committed

	return nil
}
