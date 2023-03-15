package resource

import (
	"context"
	"fmt"
	"reflect"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

// Name is a name that globally uniquely identifies a resource instance of a given type.
// Eg: for File type a Name would be the file absolute path such as /etc/issue.
type Name string

// ManageableResource defines a common interface for managing resource state.
type ManageableResource interface {
	// ValidateName validates the name of the resource
	ValidateName(name Name) error

	// GetState gets the full state of the resource.
	// If resource is not present, then returns nil.
	GetState(ctx context.Context, hst host.Host, name Name) (State, error)

	// DiffStates compares the desired State against current State.
	// If current State is met by desired State, return an empty slice; otherwise,
	// return the Diff from current State to desired State showing what needs change.
	DiffStates(
		ctx context.Context, hst host.Host,
		desiredState State, currentState State,
	) ([]diffmatchpatch.Diff, error)
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

// IndividuallyManageableResource is an interface for managing a single resource name.
// This is the most common use case, where resources can be individually managed without one resource
// having dependency on others and changing one resource does not affect the state of another.
type IndividuallyManageableResource interface {
	ManageableResource

	// Apply configures the resource to given state.
	// Must be idempotent.
	Apply(ctx context.Context, hst host.Host, name Name, state State) error

	// Destroy a configured resource at given host.
	// Must be idempotent.
	Destroy(ctx context.Context, hst host.Host, name Name) error
}

// MergeableManageableResources is an interface for managing multiple resources together.
// The use cases for this are resources where there's inter-dependency between resources, and they
// must be managed "all or nothing". The typical use case is Linux distribution package management,
// where one package may conflict with another, and the transaction of the final state must be
// computed altogether.
type MergeableManageableResources interface {
	ManageableResource

	// ConfigureAll configures all resource to given state.
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

func NewTypeFromManageableResource(manageableResource ManageableResource) Type {
	return Type(reflect.TypeOf(manageableResource).Name())
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
	state := reflect.New(reflect.TypeOf(stateInstance)).Interface().(State)
	if unmarshalSchema.Destroy {
		if unmarshalSchema.StateNode.Content != nil {
			return fmt.Errorf("line %d: can not set state when destroy is set", node.Line)
		}
	} else {
		err := unmarshalSchema.StateNode.Decode(state)
		if err != nil {
			return fmt.Errorf("line %d: %w", unmarshalSchema.StateNode.Line, err)
		}
		if err := state.Validate(); err != nil {
			return fmt.Errorf("line %d: %w", unmarshalSchema.StateNode.Line, err)
		}
	}

	*r = NewResource(
		unmarshalSchema.TypeName,
		reflect.ValueOf(state).Elem().Interface().(State),
		unmarshalSchema.Destroy,
	)
	return nil
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

// CheckState checks whether the resource is at the desired State.
// When changes are pending returns true, otherwise false.
// Diff is always returned (with or without changes).
// The current State is always returned.
func (r Resource) CheckState(
	ctx context.Context,
	hst host.Host,
	currentStatePtr *State,
) (bool, []diffmatchpatch.Diff, State, error) {
	logger := log.GetLogger(ctx)

	var currentState State
	if currentStatePtr == nil {
		var err error
		currentState, err = r.ManageableResource().GetState(ctx, hst, r.TypeName.Name())
		if err != nil {
			logger.Errorf("💥%s", r)
			return false, []diffmatchpatch.Diff{}, nil, err
		}
	} else {
		currentState = *currentStatePtr
	}

	if currentState == nil {
		if r.State != nil {
			return true, Diff(nil, r.State), nil, nil
		} else {
			return false, []diffmatchpatch.Diff{}, nil, nil
		}
	}

	if r.State == nil {
		return true, Diff(currentState, nil), nil, nil
	}
	diffs, err := r.ManageableResource().DiffStates(ctx, hst, r.State, currentState)
	if err != nil {
		logger.Errorf("💥%s", r)
		return false, []diffmatchpatch.Diff{}, nil, err
	}

	if DiffsHasChanges(diffs) {
		diffMatchPatch := diffmatchpatch.New()
		logger.WithField("", diffMatchPatch.DiffPrettyText(diffs)).
			Errorf("%s", r)
		return true, diffs, currentState, nil
	} else {
		logger.Infof("✅%s", r)
		return false, diffs, currentState, nil
	}
}

func NewResource(typeName TypeName, state State, destroy bool) Resource {
	return Resource{
		TypeName: typeName,
		State:    state,
		Destroy:  destroy,
	}
}
