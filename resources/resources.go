package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/internal/diff"
)

// Resource defines a single resource state.
// It must be marshallable.
type Resource interface {
	// Validate the state. Invalid states are things that can't exist, such as a invalid file
	// permissions, bad package name etc.
	Validate() error
}

// ValidateResource wraps Resource.Validate() with some extra common validations.
func ValidateResource(resource Resource) error {
	if GetResourceId(resource) == "" {
		resourceValue := reflect.ValueOf(resource).Elem()
		i := getResourceIdFieldIndex(resourceValue.Type())
		fieldValue := resourceValue.FieldByIndex([]int{i})
		if fieldValue.String() == "" {
			return fmt.Errorf(
				"resource id field %#v must be set",
				strings.Split(reflect.TypeOf(resource).Elem().Field(i).Tag.Get("yaml"), ",")[0],
			)
		}
	}
	if GetResourceRemove(resource) {
		removeResource := NewResourceWithSameId(resource)
		SetResourceRemove(removeResource)
		if !reflect.DeepEqual(removeResource, resource) {
			return fmt.Errorf(
				"resource has remove set to true, but other fields are set:\n%s",
				diff.DiffAsYaml(removeResource, resource).String(),
			)
		}
	}
	return resource.Validate()
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

func getResourceRemoveFieldIndex(resourceType reflect.Type) int {
	var idIndex int = -1
	for i := 0; i < resourceType.NumField(); i++ {
		structField := resourceType.FieldByIndex([]int{i})

		if structField.Type.Kind() != reflect.Bool {
			continue
		}

		if structField.Name != "Remove" {
			continue
		}

		value, ok := structField.Tag.Lookup("yaml")
		if ok {
			if value != "remove,omitempty" {
				panic(fmt.Sprintf(
					`bug: %s must tag field %s with yaml:"remove,omitempty": got yaml:"%s"`,
					resourceType.Name(), structField.Name, value,
				))
			}
		}

		idIndex = i
		break
	}
	if idIndex >= 0 {
		return idIndex
	}
	panic(fmt.Sprintf("bug: %s does not have a Remove bool field", resourceType.Name()))
}

// validateResourceStructTagYaml helps enforce that all exported fields, have a yaml tag with the
// omitempty flag set (except for the resonance:"id" tagged field).
// This is important, as it enables leaner / clearer diffs between Resource objects.
func validateResourceStructTagYaml(resourceType reflect.Type) {
	resourceIdFieldIndex := getResourceIdFieldIndex(resourceType)
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
		panic("bug: Resource Type Kind must be Struct")
	}

	// Validates the id field is tagged
	getResourceIdFieldIndex(resourceType)

	// Validate the Remove field existence
	getResourceRemoveFieldIndex(resourceType)

	validateResourceStructTagYaml(resourceType)
}

// GetResourceId returns the id for the given Resource. The id is defined as a single
// Resource struct sfield with a tag resonance:"id". The value of this id uniquely identifies
// the resource among the same type at the same host. Eg: for file, the absolute path is the id.
func GetResourceId(resource Resource) string {
	resourceValue := reflect.ValueOf(resource).Elem()
	i := getResourceIdFieldIndex(resourceValue.Type())
	fieldValue := resourceValue.FieldByIndex([]int{i})
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

	sort.Strings(names)

	return names
}

// NewResourceWithSameId copiess the given resource, and return
func NewResourceWithSameId(resource Resource) Resource {
	value := reflect.New(reflect.TypeOf(resource).Elem())
	idx := getResourceIdFieldIndex(value.Type().Elem())
	value.Elem().Field(idx).SetString(GetResourceId(resource))
	return value.Interface().(Resource)
}

// SetResourceRemove sets the given Resource Remove bool field to true.
func SetResourceRemove(resource Resource) Resource {
	value := reflect.ValueOf(resource)
	idx := getResourceRemoveFieldIndex(value.Type().Elem())
	value.Elem().Field(idx).SetBool(true)
	return value.Interface().(Resource)
}

// GetResourceRemove gets the value of the field Remove bool from given resource
func GetResourceRemove(resource Resource) bool {
	value := reflect.ValueOf(resource)
	idx := getResourceRemoveFieldIndex(value.Type().Elem())
	return value.Elem().Field(idx).Bool()
}

// ResourceMap holds references to various Resource for fast query.
type ResourceMap map[string]map[string]Resource

func NewResourceMap(resources Resources) ResourceMap {
	m := ResourceMap{}

	for _, resource := range resources {
		typeName := GetResourceTypeName(resource)
		_, ok := m[typeName]
		if !ok {
			m[typeName] = map[string]Resource{}
		}
		id := GetResourceId(resource)
		m[typeName][id] = resource
	}

	return m
}

// GetResourceWithSameId returns the Resource of the same type and id of the given resource,
// or nil if not present.
func (m ResourceMap) GetResourceWithSameTypeId(resource Resource) Resource {
	idMap, ok := m[GetResourceTypeName(resource)]
	if !ok {
		return nil
	}
	r, ok := idMap[GetResourceId(resource)]
	if !ok {
		return nil
	}
	return r
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

		if err := ValidateResource(resource); err != nil {
			return fmt.Errorf("resource %s is invalid: %s", typeName, err.Error())
		}
	}

	return nil
}

func (r Resources) MarshalYAML() (interface{}, error) {
	type MarshalSchema []map[string]Resource

	resources := make(MarshalSchema, len(r))

	for i, resource := range r {
		resourceMap := map[string]Resource{}
		typeName := reflect.TypeOf(resource).Elem().Name()
		resourceMap[typeName] = resource
		resources[i] = resourceMap
	}

	return resources, nil
}

func (r *Resources) UnmarshalYAML(node *yaml.Node) error {
	type UnmarshalSchema []map[string]yaml.Node

	resources := UnmarshalSchema{}

	node.KnownFields(true)
	err := node.Decode(&resources)
	if err != nil {
		return fmt.Errorf("line %d: %s", node.Line, err.Error())
	}

	*r = make(Resources, len(resources))

	for i, m := range resources {
		if len(m) != 1 {
			return errors.New("YAML contents does not reflect schema (bug?)")
		}
		for typeName, node := range m {
			resource := GetResourceByTypeName(typeName)
			node.KnownFields(true)
			err := node.Decode(resource)
			if err != nil {
				return fmt.Errorf("line %d: %s", node.Line, err.Error())
			}
			(*r)[i] = resource
		}
	}

	return nil
}

// NewResourcesWithSameIds is analog to NewResourceCopyWithOnlyId
func NewResourcesWithSameIds(resources Resources) Resources {
	nr := make(Resources, len(resources))

	for i, r := range resources {
		nr[i] = NewResourceWithSameId(r)
	}

	return nr
}

// GetResourcesYaml returns a string representing the resource as Yaml.
func GetResourcesYaml(resources Resources) string {
	bs, err := yaml.Marshal(resources)
	if err != nil {
		panic(err)
	}
	return strings.Trim(string(bs), "\n")
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

// GetGroupResourceTypeName returns the type name of a Resource.
func GetGroupResourceTypeName(groupResource GroupResource) string {
	return reflect.TypeOf(groupResource).Elem().Name()
}

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
