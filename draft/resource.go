package draft

import (
	"context"

	"github.com/fornellas/resonance/host/types"
)

type Resource interface {
	// Load the full resource state from Host
	Load(ctx context.Context, host types.Host) error
	// Satisfies return true when other state is satisfied by self.
	Satisfies(ctx context.Context, host types.Host, other Resource) (bool, error)
	// PreRequireFiles(ctx context.Context, host types.Host)
	// ConflictFiles(ctx context.Context, host types.Host)
	// Applies the resource state to Host.
	Apply(ctx context.Context, host types.Host) error
	// Merge attempts to merge the state of other into self. If this is not possible (eg: states
	// conflict), then it returns error, and the state of self is not altered.
	Merge(other Resource) error
	// Returns an optional list of shell file name patterns (see `path/filepath.Match`) that must
	// be applied before this resource is applied. Eg: updating an apt package must be done after
	// apt configuration (such as keys, repos etc) is applied.
	PreRequireFiles(ctx context.Context, host types.Host) []string
	// Returns an optional list of shell file name patterns (see `path/filepath.Match`) that
	// conflict with this resource, meaning that no file that matches this pattern can be declared
	// within the same host State.
	ConflictFiles(ctx context.Context, host types.Host) []string
}
