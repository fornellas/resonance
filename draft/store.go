package draft

import "context"

type Store interface {
	// Retuns HostState of managed resources before the first appply happened (used for reverting
	// the resource state when it is not managed anymore).
	GetOriginalHostState(ctx context.Context) (*HostState, error)
	// Saves the original HostState for resources before first apply.
	SaveOriginalHostState(ctx context.Context, state *HostState) error
	// Get HostState of resources that were previously applied successfully.
	GetCommittedHostState(ctx context.Context) (*HostState, error)
	// Commit planned HostState (set committed to planned, remove planned).
	CommitPlannedHostState(ctx context.Context) error
	// Get HostState that's planned before applying.
	GetPlannedHostState(ctx context.Context) (*HostState, error)
	// Save HostState that's plannned before applying.
	SavePlannedHostState(ctx context.Context, state *HostState) error
}
