package resources

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/fornellas/resonance/host"
)

// State is a Type specific interface for defining resource state as configured by users.
// If nil, it means the resource is not configured (eg: file does not exist, package not installed).
type State interface {
	// ValidateAndUpdate validates and updates the state with any required information from the host.
	// Eg: transform username into UID.
	ValidateAndUpdate(ctx context.Context, hst host.Host) (State, error)
}

// Name is a name that globally uniquely identifies a resource instance of a given type.
// Eg: for File type a Name would be the file absolute path such as /etc/issue.
type Name string

type Names []Name

func (ns Names) Len() int {
	return len(ns)
}

func (ns Names) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}

func (ns Names) Less(i, j int) bool {
	return string(ns[i]) < string(ns[j])
}

func (ns Names) String() string {
	namesStr := []string{}
	for _, name := range ns {
		namesStr = append(namesStr, string(name))
	}
	sort.Strings(namesStr)
	return strings.Join(namesStr, ",")
}

// Resource defines a common interface for managing resource state.
type Resource interface {
	// ValidateName validates the name of the resource
	ValidateName(name Name) error
}

// RefreshableResource defines an interface for resources that can be refreshed.
// Refresh means updating in-memory state as a function of file changes (eg: restarting a service,
// loading iptables rules to the kernel etc.)
type RefreshableResource interface {
	Resource

	// Refresh the resource. This is typically used to update the in-memory state of a resource
	// (eg: kerner: sysctl, iptables; process: systemd service) after persistent changes are made
	// (eg: change configuration file)
	Refresh(ctx context.Context, hst host.Host, name Name) error
}

// DiffableResource defines an interface for resources to implement their own state
// diff logic.
type DiffableResource interface {
	Resource

	// Diff compares the two States. If b is satisfied by a, it returns empty Chunks. Otherwise,
	// returns the diff between a and b.
	Diff(a, b State) Chunks
}

// MergeableResources is an interface for managing multiple resources together.
// The use cases for this are resources where there's inter-dependency between resources, and they
// must be managed "all or nothing". The typical use case is Linux distribution package management,
// where one package may conflict with another, and the transaction of the final state must be
// computed altogether.
type MergeableResources interface {
	Resource

	// GetStates gets the State of all resources, or nil if not present.
	GetStates(ctx context.Context, hst host.Host, names Names) (map[Name]State, error)

	// ApplyMerged configures all resource to given State.
	// If State is nil, it means the resource is to be unconfigured (eg: for a file, remove it).
	// Must be idempotent.
	ApplyMerged(
		ctx context.Context, hst host.Host, actionNameStateMap map[Action]map[Name]State,
	) error
}

// IndividualResource is an interface for managing a single resource name.
// This is the most common use case, where resources can be individually managed without one resource
// having dependency on others and changing one resource does not affect the state of another.
type IndividualResource interface {
	Resource

	// GetState gets the state of the resource, or nil if not present.
	GetState(ctx context.Context, hst host.Host, name Name) (State, error)

	// Apply configures the resource to given State.
	// If State is nil, it means the resource is to be unconfigured (eg: for a file, remove it).
	// Must be idempotent.
	Apply(ctx context.Context, hst host.Host, name Name, state State) error
}

// Type is the name of the resource type.
type Type string

func (t Type) validate() error {
	individualResource, ok := IndividualResourceTypeMap[t]
	if ok {
		rType := reflect.TypeOf(individualResource)
		if string(t) != rType.Name() {
			panic(fmt.Errorf(
				"%s must be defined with key %s at IndividualResourceTypeMap, not %s",
				rType.Name(), rType.Name(), string(t),
			))
		}
		return nil
	}
	mergeableResources, ok := MergeableResourcesTypeMap[t]
	if ok {
		rType := reflect.TypeOf(mergeableResources)
		if string(t) != rType.Name() {
			panic(fmt.Errorf(
				"%s must be defined with key %s at MergeableResources, not %s",
				rType.Name(), rType.Name(), string(t),
			))
		}
		return nil
	}
	return fmt.Errorf("unknown resource type '%s'", t)
}

func NewTypeFromStr(tpeStr string) (Type, error) {
	tpe := Type(tpeStr)
	if err := tpe.validate(); err != nil {
		return Type(""), err
	}
	return tpe, nil
}

// Resource returns an instance for the resource type.
func (t Type) Resource() Resource {
	individualResource, ok := IndividualResourceTypeMap[t]
	if ok {
		return individualResource
	}

	mergeableResources, ok := MergeableResourcesTypeMap[t]
	if ok {
		return mergeableResources
	}

	panic(fmt.Errorf("unknown resource type '%s'", t))
}

// MustMergeableResources returns MergeableResources from Resource or
// panics if it isn't of the required type.
func (t Type) MustMergeableResources() MergeableResources {
	mergeableResources, ok := t.Resource().(MergeableResources)
	if !ok {
		panic(fmt.Errorf("%s is not MergeableResources", t))
	}
	return mergeableResources
}

// IndividualResourceTypeMap maps Type to IndividualResource.
var IndividualResourceTypeMap = map[Type]IndividualResource{}

// MergeableResourcesTypeMap maps Type to MergeableResources.
var MergeableResourcesTypeMap = map[Type]MergeableResources{}

// ResourcesStateMap maps Type to its State.
var ResourcesStateMap = map[Type]State{}
