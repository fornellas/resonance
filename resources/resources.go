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
}

func getResourceIdFieldIndex(resourceType reflect.Type) int {
	errorMsgFmt := "bug: %s does not have a single string field with a tag resonance:\"id\""

	var idIndex int = -1
	for i := 0; i < resourceType.NumField(); i++ {
		structField := resourceType.FieldByIndex([]int{i})

		if structField.Type.Kind() != reflect.String {
			continue
		}

		value, ok := structField.Tag.Lookup("resonance")
		if !ok {
			continue
		}

		if value == "id" {
			if idIndex >= 0 {
				panic(fmt.Sprintf(errorMsgFmt, resourceType.Name()))
			}
			idIndex = i
		}
	}
	if idIndex >= 0 {
		return idIndex
	}
	panic(fmt.Sprintf(errorMsgFmt, resourceType.Name()))
}

// GetResourceId returns the id for the given Resource. The id is defined as the single
// Resource struct sfield with a tag resonance:"id". The value of this id uniquely identifies
// the resource among the same type at the same host. Eg: for file, the absolute path is the id.
func GetResourceId(ressource Resource) string {
	resourceValue := reflect.ValueOf(ressource).Elem()
	i := getResourceIdFieldIndex(resourceValue.Type())
	fieldValue := resourceValue.FieldByIndex([]int{i})
	return fieldValue.String()
}

var resourceMap = map[string]reflect.Type{}

func registerResource(resourceType reflect.Type) {
	if resourceType.Kind() != reflect.Struct {
		panic("bug: Resource Type Kind must be Struct")
	}

	// Validates the id field is tagged
	getResourceIdFieldIndex(resourceType)

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
func GetResourceByTypeName(name string) Resource {
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
func GetResourceTypeNames() []string {
	names := make([]string, len(resourceMap))

	i := 0
	for name := range resourceMap {
		names[i] = name
		i++
	}

	return names
}

// NewResourceCopyWithOnlyId copiess the given resource, and return
func NewResourceCopyWithOnlyId(resource Resource) Resource {
	value := reflect.New(reflect.TypeOf(resource).Elem())
	idx := getResourceIdFieldIndex(value.Type().Elem())
	value.Elem().Field(idx).SetString(GetResourceId(resource))
	return value.Interface().(Resource)
}

type Resources []Resource

func (r Resources) Ids() string {
	ids := make([]string, len(r))
	for i, resource := range r {
		ids[i] = GetResourceId(resource)
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func (r Resources) Validate() error {
	typeNameMap := map[string]bool{}
	for _, resource := range r {
		typeName := fmt.Sprintf("%s:%s", reflect.TypeOf(resource).Name(), GetResourceId(resource))
		if _, ok := typeNameMap[typeName]; ok {
			return fmt.Errorf("duplicated resource %s", typeName)
		}

		if err := resource.Validate(); err != nil {
			return fmt.Errorf("resource %s is invalid: %s", typeName, err.Error())
		}
	}

	return nil
}

func (r Resources) MarshalYAML() (interface{}, error) {
	type ResourcesYamlSchema []map[string]Resource

	resourcesYaml := make(ResourcesYamlSchema, len(r))

	for i, resource := range r {
		resourceMap := map[string]Resource{}
		typeName := reflect.TypeOf(resource).Elem().Name()
		resourceMap[typeName] = resource
		resourcesYaml[i] = resourceMap
	}

	return resourcesYaml, nil
}

// NewResourcesCopyWithOnlyId is analog to NewResourceCopyWithOnlyId
func NewResourcesCopyWithOnlyId(resources Resources) Resources {
	nr := make(Resources, len(resources))

	for i, r := range resources {
		nr[i] = NewResourceCopyWithOnlyId(r)
	}

	return nr
}

// A SingleResource is something that can be configured independently of all resources of the same
// type. Eg: a user.
type SingleResource interface {
	Resource
	// Load the full current state of the resource from given Host.
	Load(context.Context, host.Host) error
	// Resolve the state with information that may be required from the host. This must not change
	// the semantics of the state.
	// Eg: for a file, the state defines a username, which must be transformed to a UID.
	Resolve(context.Context, host.Host) error
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
func GetSingleResourceByTypeName(name string) SingleResource {
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
	// Resolve the state with information that may be required from the host. This must not change
	// the semantics of the state.
	// Eg: for a file, the state defines a username, which must be transformed to a UID.
	Resolve(context.Context, host.Host, Resources) error
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
func GetGroupResourceByTypeName(name string) GroupResource {
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
