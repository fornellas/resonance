package resources

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/fornellas/resonance/host"
)

// Resource defines a single resource state.
// It must be marshallable.
type Resource interface {
	// Validate the state. Invalid states are things that can't exist, such as a invalid file
	// permissions, bad package name etc.
	Validate() error
	// Name that uniquely identify a resource of its type. Eg: for file, the full path uniquely
	// identifies each file, and must be its name.
	Name() string
}

var resourceMap = map[string]reflect.Type{}

func registerResource(resourceType reflect.Type) {
	typeName := resourceType.Name()

	if !(reflect.PointerTo(resourceType)).Implements(reflect.TypeOf((*Resource)(nil)).Elem()) {
		panic("bug: resourceType does not implement Resource")
	}

	if _, ok := resourceMap[typeName]; ok {
		panic(fmt.Sprintf("double registration of Resource %#v", typeName))
	}
	resourceMap[typeName] = resourceType
}

// Returns a Resource of a previously registered with RegisterSingleResource or
// RegisterGroupResource for given reflect.Type name.
func GetResourceByName(name string) Resource {
	resource, ok := resourceMap[name]
	if !ok {
		return nil
	}
	value := reflect.New(resource)
	instance, ok := value.Interface().(Resource)
	if !ok {
		panic("bug: registered resource doesn't implement Resource")
	}
	return instance
}

// Returns the list of Resource names previously registered with RegisterSingleResource or
// RegisterGroupResource.
func GetResourceNames() []string {
	names := make([]string, len(resourceMap))

	i := 0
	for name := range resourceMap {
		names[i] = name
		i++
	}

	return names
}

type Resources []Resource

func (r Resources) Names() string {
	names := make([]string, len(r))
	for i, resource := range r {
		names[i] = resource.Name()
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

func (r Resources) Validate() error {
	typeNameMap := map[string]bool{}
	for _, resource := range r {
		typeName := fmt.Sprintf("%s:%s", reflect.TypeOf(resource).Name(), resource.Name())
		if _, ok := typeNameMap[typeName]; ok {
			return fmt.Errorf("duplicated resource %s", typeName)
		}

		if err := resource.Validate(); err != nil {
			return fmt.Errorf("resource %s is invalid: %s", typeName, err.Error())
		}
	}

	return nil
}

// A SingleResource is something that can be configured independently of all resources of the same
// type. Eg: a user.
type SingleResource interface {
	Resource
	// Load the full current state of the resource from given Host.
	Load(context.Context, host.Host) error
	// Updates the state with information that may be required from the host. This must not change
	// the semantics of the state.
	// Eg: for a file, the state defines a username, which must be transformed to a UID.
	Update(context.Context, host.Host) error
	// Apply state of the resource to given Host.
	Apply(context.Context, host.Host) error
}

// Register a new SingleResource type.
func RegisterSingleResource(singleResourceType reflect.Type) {
	if !(reflect.PointerTo(singleResourceType)).Implements(reflect.TypeOf((*SingleResource)(nil)).Elem()) {
		panic("bug: singleResourceType does not implement SingleResource")
	}
	registerResource(singleResourceType)
}

// Returns a SingleResource of a previously registered with RegisterSingleResource for
// given reflect.Type name.
func GetSingleResourceByName(name string) SingleResource {
	singleResourceType, ok := resourceMap[name]
	if !ok {
		return nil
	}
	value := reflect.New(singleResourceType)
	instance, ok := value.Interface().(SingleResource)
	if !ok {
		panic("bug: registered resource doesn't implement SingleResource")
	}
	return instance
}

// GroupResource implements how to configure resources which have inter-dependency within the same
// type. Eg: APT packages may conflict with each other, so they must be configured altogether.
type GroupResource interface {
	// Load the full current state of all resources from given Host.
	Load(context.Context, host.Host, Resources) error
	// Updates the state with information that may be required from the host. This must not change
	// the semantics of the state.
	// Eg: for a file, the state defines a username, which must be transformed to a UID.
	Update(context.Context, host.Host, Resources) error
	// Apply state of all given resources to Host.
	Apply(context.Context, host.Host, Resources) error
}

var groupResourceMap = map[string]reflect.Type{}

// Register a group resource. Such resources have inter-dependency among the same type and must be
// configured altogether. You must pass a Resource type (which holds the state of each resource) and
// a GroupResource type, which handles how to configured such resources altogether.
func RegisterGroupResource(resourceType, groupResourceType reflect.Type) {
	registerResource(resourceType)

	if !(reflect.PointerTo(groupResourceType)).Implements(reflect.TypeOf((*GroupResource)(nil)).Elem()) {
		panic("bug: groupResourceType does not implement GroupResource")
	}
	groupResourceMap[resourceType.Name()] = groupResourceType
}

// Whether a resource, previously registered with either RegisterSingleResource or
// RegisterGroupResource is a group resource.
func IsGroupResource(name string) bool {
	_, ok := groupResourceMap[name]
	return ok
}

// Returns a GroupResource of a previously registered with RegisterGroupResource for
// given reflect.Type name.
func GetGroupResourceByName(name string) GroupResource {
	groupResourceType, ok := groupResourceMap[name]
	if !ok {
		return nil
	}
	value := reflect.New(groupResourceType)
	instance, ok := value.Interface().(GroupResource)
	if !ok {
		panic("bug: registered resource doesn't implement GroupResource")
	}
	return instance
}
