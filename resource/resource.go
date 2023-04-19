package resource

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
)

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

// ManageableResource defines a common interface for managing resource state.
type ManageableResource interface {
	// ValidateName validates the name of the resource
	ValidateName(name Name) error
}

// RefreshableManageableResource defines an interface for resources that can be refreshed.
// Refresh means updating in-memory state as a function of file changes (eg: restarting a service,
// loading iptables rules to the kernel etc.)
type RefreshableManageableResource interface {
	ManageableResource

	// Refresh the resource. This is typically used to update the in-memory state of a resource
	// (eg: kerner: sysctl, iptables; process: systemd service) after persistent changes are made
	// (eg: change configuration file)
	Refresh(ctx context.Context, hst host.Host, name Name) error
}

// DiffableManageableResource defines an interface for resources to implement their own state
// diff logic.
type DiffableManageableResource interface {
	ManageableResource

	// Diff compares the two States. If b is satisfied by a, it returns empty Chunks. Otherwise,
	// returns the diff between a and b.
	Diff(a, b State) Chunks
}

// IndividuallyManageableResource is an interface for managing a single resource name.
// This is the most common use case, where resources can be individually managed without one resource
// having dependency on others and changing one resource does not affect the state of another.
type IndividuallyManageableResource interface {
	ManageableResource

	// GetState gets the state of the resource, or nil if not present.
	GetState(ctx context.Context, hst host.Host, name Name) (State, error)

	// Configure configures the resource to given State.
	// If State is nil, it means the resource is to be unconfigured (eg: for a file, remove it).
	// Must be idempotent.
	Configure(ctx context.Context, hst host.Host, name Name, state State) error
}

// MergeableManageableResources is an interface for managing multiple resources together.
// The use cases for this are resources where there's inter-dependency between resources, and they
// must be managed "all or nothing". The typical use case is Linux distribution package management,
// where one package may conflict with another, and the transaction of the final state must be
// computed altogether.
type MergeableManageableResources interface {
	ManageableResource

	// GetStates gets the State of all resources, or nil if not present.
	GetStates(ctx context.Context, hst host.Host, names Names) (map[Name]State, error)

	// ConfigureAll configures all resource to given State.
	// If State is nil, it means the resource is to be unconfigured (eg: for a file, remove it).
	// Must be idempotent.
	ConfigureAll(
		ctx context.Context, hst host.Host, actionNameStateMap map[Action]map[Name]State,
	) error
}

// Type is the name of the resource type.
type Type string

func (t Type) validate() error {
	individuallyManageableResource, ok := IndividuallyManageableResourceTypeMap[t]
	if ok {
		rType := reflect.TypeOf(individuallyManageableResource)
		if string(t) != rType.Name() {
			panic(fmt.Errorf(
				"%s must be defined with key %s at IndividuallyManageableResourceTypeMap, not %s",
				rType.Name(), rType.Name(), string(t),
			))
		}
		return nil
	}
	mergeableManageableResources, ok := MergeableManageableResourcesTypeMap[t]
	if ok {
		rType := reflect.TypeOf(mergeableManageableResources)
		if string(t) != rType.Name() {
			panic(fmt.Errorf(
				"%s must be defined with key %s at MergeableManageableResources, not %s",
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

// ManageableResource returns an instance for the resource type.
func (t Type) ManageableResource() ManageableResource {
	individuallyManageableResource, ok := IndividuallyManageableResourceTypeMap[t]
	if ok {
		return individuallyManageableResource
	}

	mergeableManageableResources, ok := MergeableManageableResourcesTypeMap[t]
	if ok {
		return mergeableManageableResources
	}

	panic(fmt.Errorf("unknown resource type '%s'", t))
}

// MustMergeableManageableResources returns MergeableManageableResources from ManageableResource or
// panics if it isn't of the required type.
func (t Type) MustMergeableManageableResources() MergeableManageableResources {
	mergeableManageableResources, ok := t.ManageableResource().(MergeableManageableResources)
	if !ok {
		panic(fmt.Errorf("%s is not MergeableManageableResources", t))
	}
	return mergeableManageableResources
}

// IndividuallyManageableResourceTypeMap maps Type to IndividuallyManageableResource.
var IndividuallyManageableResourceTypeMap = map[Type]IndividuallyManageableResource{}

// MergeableManageableResourcesTypeMap maps Type to MergeableManageableResources.
var MergeableManageableResourcesTypeMap = map[Type]MergeableManageableResources{}

// ManageableResourcesStateMap maps Type to its State.
var ManageableResourcesStateMap = map[Type]State{}

// TypeName is a string that identifies a resource type and name.
// Eg: File[/etc/issue].
type TypeName string

var resourceInstanceKeyRegexp = regexp.MustCompile(`^(.+)\[(.+)\]$`)

func (tn TypeName) typeName() (Type, Name, error) {
	var tpe Type
	var name Name
	matches := resourceInstanceKeyRegexp.FindStringSubmatch(string(tn))
	if len(matches) != 3 {
		return tpe, name, fmt.Errorf("%#v does not match Type[Name] format", tn)
	}
	tpe, err := NewTypeFromStr(matches[1])
	if err != nil {
		return Type(""), Name(""), err
	}
	name = Name(matches[2])
	return tpe, name, nil
}

func (tn *TypeName) UnmarshalYAML(node *yaml.Node) error {
	var typeNameStr string
	node.KnownFields(true)
	if err := node.Decode(&typeNameStr); err != nil {
		return err
	}
	typeName, err := NewTypeNameFromStr(typeNameStr)
	if err != nil {
		return err
	}
	*tn = typeName
	return nil
}

func (tn TypeName) Type() Type {
	tpe, _, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return tpe
}

func (tn TypeName) Name() Name {
	_, name, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return name
}

// ManageableResource returns an instance for the resource type.
func (tn TypeName) ManageableResource() ManageableResource {
	tpe, _, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return tpe.ManageableResource()
}

// IsIndividuallyManageableResource returns true if ManageableResource() is of type IndividuallyManageableResource.
func (tn TypeName) IsIndividuallyManageableResource() bool {
	_, ok := tn.ManageableResource().(IndividuallyManageableResource)
	return ok
}

// MustIndividuallyManageableResource returns IndividuallyManageableResource from ManageableResource or
// panics if it isn't of the required type.
func (tn TypeName) MustIndividuallyManageableResource() IndividuallyManageableResource {
	individuallyManageableResource, ok := tn.ManageableResource().(IndividuallyManageableResource)
	if !ok {
		panic(fmt.Errorf("%s is not IndividuallyManageableResource", tn))
	}
	return individuallyManageableResource
}

// IsMergeableManageableResources returns true only if ManageableResource() is of type MergeableManageableResources.
func (tn TypeName) IsMergeableManageableResources() bool {
	_, ok := tn.ManageableResource().(MergeableManageableResources)
	return ok
}

func NewTypeName(tpe Type, name Name) (TypeName, error) {
	return NewTypeNameFromStr(fmt.Sprintf("%s[%s]", tpe, name))
}

func MustNewTypeName(tpe Type, name Name) TypeName {
	typeName, err := NewTypeNameFromStr(fmt.Sprintf("%s[%s]", tpe, name))
	if err != nil {
		panic(err)
	}
	return typeName
}

func NewTypeNameFromStr(typeNameStr string) (TypeName, error) {
	typeName := TypeName(typeNameStr)
	_, _, err := typeName.typeName()
	if err != nil {
		return TypeName(""), err
	}
	return typeName, nil
}

// Resource holds a single resource.
type Resource struct {
	TypeName TypeName `yaml:"resource"`
	State    State    `yaml:"state"`
	Destroy  bool     `yaml:"destroy"`
}

type resourceUnmarshalSchema struct {
	TypeName  TypeName  `yaml:"resource"`
	StateNode yaml.Node `yaml:"state"`
	Destroy   bool      `yaml:"destroy"`
}

func (r *Resource) UnmarshalYAML(node *yaml.Node) error {
	var unmarshalSchema resourceUnmarshalSchema
	node.KnownFields(true)
	if err := node.Decode(&unmarshalSchema); err != nil {
		return err
	}

	manageableResource := unmarshalSchema.TypeName.ManageableResource()
	tpe := unmarshalSchema.TypeName.Type()
	name := unmarshalSchema.TypeName.Name()
	if err := manageableResource.ValidateName(name); err != nil {
		return fmt.Errorf("line %d: %w", node.Line, err)
	}

	stateInstance, ok := ManageableResourcesStateMap[tpe]
	if !ok {
		panic(fmt.Errorf("Type %s missing from ManageableResourcesStateMap", tpe))
	}
	var state State
	if unmarshalSchema.Destroy {
		if unmarshalSchema.StateNode.Content != nil {
			return fmt.Errorf("line %d: can not set state when destroy is set", node.Line)
		}
	} else {
		stateInstance := reflect.New(reflect.TypeOf(stateInstance)).Interface().(State)
		err := unmarshalSchema.StateNode.Decode(stateInstance)
		if err != nil {
			return fmt.Errorf("line %d: %w", unmarshalSchema.StateNode.Line, err)
		}
		state = reflect.ValueOf(stateInstance).Elem().Interface().(State)
	}

	*r = NewResource(
		unmarshalSchema.TypeName,
		state,
		unmarshalSchema.Destroy,
	)
	return nil
}

func (r Resource) MarshalYAML() (interface{}, error) {
	if r.Destroy {
		r.State = nil
	}
	type resourceAlias Resource
	node := yaml.Node{}
	err := node.Encode(resourceAlias(r))
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (r Resource) MustType() Type {
	return r.TypeName.Type()
}

func (r Resource) MustName() Name {
	return r.TypeName.Name()
}

func (r Resource) String() string {
	return string(r.TypeName)
}

func (r Resource) ManageableResource() ManageableResource {
	return r.TypeName.ManageableResource()
}

// Refreshable returns whether the resource is refreshable or not.
func (r Resource) Refreshable() bool {
	_, ok := r.ManageableResource().(RefreshableManageableResource)
	return ok
}

// MustIndividuallyManageableResource returns IndividuallyManageableResource from ManageableResource or
// panics if it isn't of the required type.
func (r Resource) MustIndividuallyManageableResource() IndividuallyManageableResource {
	individuallyManageableResource, ok := r.ManageableResource().(IndividuallyManageableResource)
	if !ok {
		panic(fmt.Errorf("%s is not IndividuallyManageableResource", r))
	}
	return individuallyManageableResource
}

// IsMergeableManageableResources returns true only if ManageableResource is of type MergeableManageableResources.
func (r Resource) IsMergeableManageableResources() bool {
	_, ok := r.ManageableResource().(MergeableManageableResources)
	return ok
}

// MustMergeableManageableResources returns MergeableManageableResources from ManageableResource or
// panics if it isn't of the required type.
func (r Resource) MustMergeableManageableResources() MergeableManageableResources {
	mergeableManageableResources, ok := r.ManageableResource().(MergeableManageableResources)
	if !ok {
		panic(fmt.Errorf("%s is not MergeableManageableResources", r))
	}
	return mergeableManageableResources
}

func NewResource(typeName TypeName, state State, destroy bool) Resource {
	return Resource{
		TypeName: typeName,
		State:    state,
		Destroy:  destroy,
	}
}
