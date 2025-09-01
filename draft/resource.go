package draft

import (
	"context"

	"github.com/fornellas/resonance/host/types"
)

// Resource holds the definition for an individual resource.
type Resource interface {
	// ID uniquely identifies the resource at a host.
	ID() string
	// Satisfies returns true when self satisfies the state required by other.
	Satisfies(ctx context.Context, host types.Host, otherResource Resource) (bool, error)
	// Validates whether the resource state is valid (eg: a file has a non absolute path as an error).
	Validate() error
}

// Resource enables managing (load, apply...) a single resource of group of related resources.
// type Resource interface {
// 	// Load the full resource state from Host.
// 	Load(ctx context.Context, host types.Host) (Resource, error) // FIXME can't return Resource
// 	// Applies the resource state to Host.
// 	Apply(ctx context.Context, host types.Host) error
// 	// Merge attempts to merge the state of other into self. If this is not possible (eg: states
// 	// conflict), then it returns error, and the state of self is not altered.
// 	// Merge(other Resource) error
// 	// Returns an optional list of shell file name patterns (see `path/filepath.Match`) that must
// 	// be applied before this resource is applied. Eg: updating an apt package must be done after
// 	// apt configuration (such as keys, repos etc) is applied.
// 	// PreRequireFiles(ctx context.Context, host types.Host) []string
// 	// Returns an optional list of shell file name patterns (see `path/filepath.Match`) that
// 	// conflict with this resource, meaning that no file that matches this pattern can be declared
// 	// within the same host State.
// 	// ConflictFiles(ctx context.Context, host types.Host) []string
// }
