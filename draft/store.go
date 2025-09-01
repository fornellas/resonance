package draft

import "context"

type Store interface {
	// Retuns State of managed resources before the first appply happened (used for reverting
	// the resource state when it is not managed anymore).
	GetOriginalState(ctx context.Context) (*State, error)
	// Saves the original State for resources before first apply.
	SaveOriginalState(ctx context.Context, state *State) error
	// Get State of resources that were previously applied successfully.
	GetCommittedState(ctx context.Context) (*State, error)
	// Commit planned State (set committed to planned, remove planned).
	CommitPlannedState(ctx context.Context) error
	// Get State that's planned before applying.
	GetPlannedState(ctx context.Context) (*State, error)
	// Save State that's plannned before applying.
	SavePlannedState(ctx context.Context, state *State) error
}
