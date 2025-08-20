// Host state definition storage utilities.
package store

import (
	"context"
	"io"
)

// Store defines an interface for storage of host state.
type Store interface {
	// GetLogWriterCloser returns a io.WriteCloser to be used for logging for the current session,
	// with given name. On session completion, the object must be closed.
	// The implementation is responsible for doing log rotation and purge when this function is
	// called.
	GetLogWriterCloser(ctx context.Context, name string) (io.WriteCloser, error)
}
