// Host state definition storage utilities.
package store

import (
	"context"
	"io"

	"github.com/fornellas/resonance/state"
)

// Store defines an interface for storage of host state.
type Store interface {
	// Returns the State of managed resources before the first appply happened (used for reverting
	// the resource state when it is not managed anymore).
	GetOriginalState(ctx context.Context) (*state.State, error)
	// Saves the original State for resources before first apply.
	SaveOriginalState(ctx context.Context, state *state.State) error
	// Get the State of resources that were previously applied successfully.
	GetCommittedState(ctx context.Context) (*state.State, error)
	// Commit planned State (set committed to planned, remove planned).
	CommitPlannedState(ctx context.Context) error
	// Get State that's planned before applying.
	GetPlannedState(ctx context.Context) (*state.State, error)
	// Save State that's plannned before applying.
	SavePlannedState(ctx context.Context, state *state.State) error
	// GetLogWriterCloser returns a io.WriteCloser to be used for logging for the current session,
	// with given name. On session completion, the object must be closed.
	// The implementation is responsible for doing log rotation and purge when this function is
	// called.
	GetLogWriterCloser(ctx context.Context, name string) (io.WriteCloser, error)
}
