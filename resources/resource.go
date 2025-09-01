package resources

import (
	"context"

	"github.com/fornellas/resonance/host/types"
)

// Resource holds the state definition for an individual resource. This state may be partial (eg:
// just the file owner, not its contents) or complete (all file attributes).
type Resource interface {
	// ID uniquely identifies the resource at a host.
	ID() string
	// Satisfies returns true when self satisfies the state required by other. Useful to tell
	// when a full state (eg: fetched from host) satisfies a partial state (provided as input).
	Satisfies(ctx context.Context, host types.Host, otherResource Resource) (bool, error)
	// Validates whether the resource state is valid (eg: a file has a non absolute path as an
	// error).
	Validate() error
	// Merge attempts to merge the state of other into self. If this is not possible (eg: states
	// conflict), then it returns error, and the state of self is not altered.
	Merge(other Resource) error
}

// // ManagedResource manages the state of a single resource of a group of same type resources at
// // a host.
// type ManagedResource interface {
// 	// Load the full resource state from Host.
// 	Load(ctx context.Context, host types.Host) (ManagedResource, error)
// 	// Applies the resource state to Host.
// 	Apply(ctx context.Context, host types.Host) error
// 	// Returns an optional list of shell file name patterns (see `path/filepath.Match`) that must
// 	// be applied before this resource is applied. Eg: updating an apt package must be done after
// 	// apt configuration (such as keys, repos etc) is applied.
// 	PreRequireFiles(ctx context.Context, host types.Host) []string
// 	// Returns an optional list of shell file name patterns (see `path/filepath.Match`) that
// 	// conflict with this resource, meaning that no file that matches this pattern can be declared
// 	// within the same host State.
// 	ConflictFiles(ctx context.Context, host types.Host) []string
// }
