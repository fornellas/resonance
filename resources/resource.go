package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/diff"
	"github.com/fornellas/resonance/host/types"
)

////////////////////////////////////////////////////////////////////////////////////////////////////
// Resource
////////////////////////////////////////////////////////////////////////////////////////////////////

// Resource defines a single resource state.
// It must be marshallable.
type Resource interface {
	// Validate the state. Invalid states are things that can't exist, such as a invalid file
	// permissions, bad package name etc.
	Validate() error
}

// Resource types are expected to be of Kind Struct, and have the first field as string
var resourceIdFieldIndex = 0

// Resource types are expected to be of Kind Struct, and have the second field as bool, to indicate
// whether the resource is absent or not.
var resourceAbsentFieldIndex = 1

// validateResourceStructTagYaml helps enforce that all exported fields, have a yaml tag with the
// omitempty flag set (except for the resonance:"id" tagged field).
// This is important, as it enables leaner / clearer diffs between Resource objects.
func validateResourceStructTagYaml(resourceType reflect.Type) {
	for i := 0; i < resourceType.NumField(); i++ {
		structField := resourceType.FieldByIndex([]int{i})

		if len(structField.Name) < 1 {
			continue
		}

		if !unicode.IsUpper(rune(structField.Name[0])) {
			continue
		}

		value, ok := structField.Tag.Lookup("yaml")
		if !ok {
			if resourceIdFieldIndex == i {
				continue
			}
			panic(fmt.Sprintf(
				`bug: %s must tag field %s with yaml:"*,omitempty"`,
				resourceType.Name(), structField.Name,
			))
		}

		values := strings.Split(value, ",")
		if len(values) < 2 {
			if resourceIdFieldIndex == i {
				continue
			}
			panic(fmt.Sprintf(
				`bug: %s must tag field %s with yaml:"*,omitempty", got: yaml"%s"`,
				resourceType.Name(), structField.Name, value,
			))
		}

		hasOmitempty := false
		for _, flag := range values[1:] {
			if flag == "omitempty" {
				hasOmitempty = true
				break
			}
		}
		if hasOmitempty {
			if resourceIdFieldIndex == i {
				panic(fmt.Sprintf(
					`bug: %s field %s is tagged with resonance:"id", it can not be tagged with yaml:"*,omitempty"; got: yaml"%s"`,
					resourceType.Name(), structField.Name, value,
				))
			}
		} else {
			panic(fmt.Sprintf(
				`bug: %s must tag field %s with yaml:"*,omitempty", got: yaml"%s"`,
				resourceType.Name(), structField.Name, value,
			))
		}
	}
}

// validateResourceStruct helps enforce that all Resource types are Structs and that that
// they have required fields and tags.
func validateResourceStruct(resourceType reflect.Type) {
	if resourceType.Kind() != reflect.Struct {
		panic(fmt.Sprintf(
			"bug: resource %#v must be of Kind struct, got %#v", resourceType.Name(), resourceType.Kind()),
		)
	}

	if resourceType.NumField() < 2 {
		panic(fmt.Sprintf("resource %s must have at least 2 fields: one Id first and Absent bool second ", resourceType.Name()))
	}

	// Id
	idStructField := resourceType.FieldByIndex([]int{resourceIdFieldIndex})
	if idStructField.Type.Kind() != reflect.String {
		panic(fmt.Sprintf(
			"bug: resource %#v must have first field of type string, to uniquely identify the resource, got %#v",
			resourceType.Name(), idStructField.Type.Kind(),
		))
	}

	// Absent
	absentStructField := resourceType.FieldByIndex([]int{resourceAbsentFieldIndex})
	if absentStructField.Type.Kind() != reflect.Bool {
		panic(fmt.Sprintf(
			"bug: resource %#v must have second field of type bool, to indicate resource absence",
			resourceType.Name(),
		))
	}
	if absentStructField.Name != "Absent" {
		panic(fmt.Sprintf(
			"bug: resource %#v must have second field named Absent",
			resourceType.Name(),
		))
	}
	value, ok := absentStructField.Tag.Lookup("yaml")
	if ok {
		if value != "absent,omitempty" {
			panic(fmt.Sprintf(
				`bug: %#v must tag field Absent with yaml:"absent,omitempty": got yaml:"%s"`,
				resourceType.Name(), value,
			))
		}
	}

	validateResourceStructTagYaml(resourceType)
}

var resourceMap = map[string]reflect.Type{}

func registerResource(resourceType reflect.Type) {
	validateResourceStruct(resourceType)

	typeName := resourceType.Name()

	if !(reflect.PointerTo(resourceType)).Implements(reflect.TypeOf((*Resource)(nil)).Elem()) {
		panic("bug: resourceType does not implement Resource")
	}

	if _, ok := resourceMap[typeName]; ok {
		panic(fmt.Sprintf("double registration of Resource %#v", typeName))
	}

	resourceMap[typeName] = resourceType
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

	sort.Strings(names)

	return names
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

// NewResourceWithSameId copiess the given resource, and return
func NewResourceWithSameId(resource Resource) Resource {
	value := reflect.New(reflect.TypeOf(resource).Elem())
	value.Elem().Field(resourceIdFieldIndex).SetString(GetResourceId(resource))
	return value.Interface().(Resource)
}

// GetResourceId returns the id for the given Resource. The id is defined as a single
// Resource struct sfield with a tag resonance:"id". The value of this id uniquely identifies
// the resource among the same type at the same host. Eg: for file, the absolute path is the id.
func GetResourceId(resource Resource) string {
	resourceValue := reflect.ValueOf(resource).Elem()
	fieldValue := resourceValue.FieldByIndex([]int{resourceIdFieldIndex})
	return fieldValue.String()
}

// GetResourceTypeName returns the type name of a Resource.
func GetResourceTypeName(resource Resource) string {
	return reflect.TypeOf(resource).Elem().Name()
}

// GetResourceId returnss a string Type:Id for the resource.
func GetResourceTypeId(resource Resource) string {
	return fmt.Sprintf("%s:%s", GetResourceTypeName(resource), GetResourceId(resource))
}

// GetResourceYaml returns a string representing the resource as Yaml.
func GetResourceYaml(resource Resource) string {
	bs, err := yaml.Marshal(resource)
	if err != nil {
		panic(err)
	}
	return strings.Trim(string(bs), "\n")
}

// HashResource returns a hex encoded string that is a hashed value, function of the given
// resource type and its Id
var HashResource = func(resource Resource) string {
	name := fmt.Sprintf(
		"%s:%s",
		GetResourceTypeName(resource),
		GetResourceId(resource),
	)
	hash := sha256.Sum256([]byte(name))
	return hex.EncodeToString(hash[:])
}

// ValidateResource wraps Resource.Validate() with some extra common validations.
func ValidateResource(resource Resource) error {
	if GetResourceId(resource) == "" {
		resourceValue := reflect.ValueOf(resource).Elem()
		fieldValue := resourceValue.FieldByIndex([]int{resourceIdFieldIndex})
		if fieldValue.String() == "" {
			return fmt.Errorf(
				"resource id field %#v must be set",
				strings.Split(reflect.TypeOf(resource).Elem().Field(resourceIdFieldIndex).Tag.Get("yaml"), ",")[0],
			)
		}
	}
	if GetResourceAbsent(resource) {
		absentResource := NewResourceWithSameId(resource)
		SetResourceAbsent(absentResource)
		if !reflect.DeepEqual(absentResource, resource) {
			return fmt.Errorf(
				"resource has absent set to true, but other fields are set:\n%s",
				diff.DiffAsYaml(absentResource, resource).String(),
			)
		}
	}
	return resource.Validate()
}

// SetResourceAbsent sets the given Resource Absent bool field to true.
func SetResourceAbsent(resource Resource) Resource {
	value := reflect.ValueOf(resource)
	value.Elem().Field(resourceAbsentFieldIndex).SetBool(true)
	return value.Interface().(Resource)
}

// GetResourceAbsent gets the value of the field Absent bool from given resource
func GetResourceAbsent(resource Resource) bool {
	value := reflect.ValueOf(resource)
	return value.Elem().Field(resourceAbsentFieldIndex).Bool()
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Satisfiable
////////////////////////////////////////////////////////////////////////////////////////////////////

// Satisfiable interface can be implemented by resources which can't be compared by simply comparing
// two structs, as some fields may have specific semants.
// Eg: if (a) defines a package with a name and a specific version, but
// (b) specifies a package with the same name, but with any version, then
// (a) satisfies (b), but (b) does not satisfy (a).
type Satisfiable interface {
	// Satisfies returns true only when it satisfies the given resource.
	Satisfies(resource Resource) bool
}

// Satisfies returns whether (a) satisfies (b).
// Eg: if (a) defines a package with a name and a specific version, but
// (b) specifies a package with the same name, but with any version, then
// (a) satisfies (b).
func Satisfies(a, b Resource) bool {
	aSatisfiable, ok := a.(Satisfiable)
	if !ok {
		return reflect.DeepEqual(a, b)
	}
	return aSatisfiable.Satisfies(b)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// SingleResource
////////////////////////////////////////////////////////////////////////////////////////////////////

// A SingleResource is something that can be configured independently of all resources of the same
// type. Eg: a user.
type SingleResource interface {
	Resource
	// Load the full current state of the resource from given Host.
	Load(context.Context, types.Host) error
	// Resolve the state with information that may be required from the host. This must not change
	// the semantics of the state.
	// Eg: for a file, the state defines a username, which must be transformed to a UID.
	Resolve(context.Context, types.Host) error
	// Apply state of the resource to given Host.
	Apply(context.Context, types.Host) error
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

////////////////////////////////////////////////////////////////////////////////////////////////////
// GroupResource
////////////////////////////////////////////////////////////////////////////////////////////////////

// GroupResource implements how to configure resources which have inter-dependency within the same
// type. Eg: APT packages may conflict with each other, so they must be configured altogether.
type GroupResource interface {
	// Load the full current state of all resources from given Host.
	Load(context.Context, types.Host, Resources) error
	// Resolve the state with information that may be required from the host. This must not change
	// the semantics of the state.
	// Eg: for a file, the state defines a username, which must be transformed to a UID.
	Resolve(context.Context, types.Host, Resources) error
	// Apply state of all given resources to Host.
	Apply(context.Context, types.Host, Resources) error
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

// GetGroupResourceTypeName returns the type name of a Resource.
func GetGroupResourceTypeName(groupResource GroupResource) string {
	return reflect.TypeOf(groupResource).Elem().Name()
}

// Returns a GroupResource of a previously registered with RegisterGroupResource for
// given resource type name.
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

// Whether a resource type name, previously registered with either RegisterSingleResource or
// RegisterGroupResource is a group resource.
func IsGroupResource(name string) bool {
	_, ok := groupResourceMap[name]
	return ok
}
