package resource

import (
	"fmt"
	"reflect"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/resources"
)

// TypeName is a string that identifies a resource type and name.
// Eg: File[/etc/issue].
type TypeName string

var resourceInstanceKeyRegexp = regexp.MustCompile(`^([^\[]+)\[(.+)\]$`)

func (tn TypeName) typeName() (resources.Type, resources.Name, error) {
	var tpe resources.Type
	var name resources.Name
	matches := resourceInstanceKeyRegexp.FindStringSubmatch(string(tn))
	if len(matches) != 3 {
		return tpe, name, fmt.Errorf("%#v does not match Type[Name] format", tn)
	}
	tpe, err := resources.NewTypeFromStr(matches[1])
	if err != nil {
		return resources.Type(""), resources.Name(""), err
	}
	name = resources.Name(matches[2])
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

func (tn TypeName) Type() resources.Type {
	tpe, _, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return tpe
}

func (tn TypeName) Name() resources.Name {
	_, name, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return name
}

// ManageableResource returns an instance for the resource type.
func (tn TypeName) ManageableResource() resources.ManageableResource {
	tpe, _, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return tpe.ManageableResource()
}

// IsIndividuallyManageableResource returns true if ManageableResource() is of type IndividuallyManageableResource.
func (tn TypeName) IsIndividuallyManageableResource() bool {
	_, ok := tn.ManageableResource().(resources.IndividuallyManageableResource)
	return ok
}

// MustIndividuallyManageableResource returns IndividuallyManageableResource from ManageableResource or
// panics if it isn't of the required type.
func (tn TypeName) MustIndividuallyManageableResource() resources.IndividuallyManageableResource {
	individuallyManageableResource, ok := tn.ManageableResource().(resources.IndividuallyManageableResource)
	if !ok {
		panic(fmt.Errorf("%s is not IndividuallyManageableResource", tn))
	}
	return individuallyManageableResource
}

// IsMergeableManageableResources returns true only if ManageableResource() is of type MergeableManageableResources.
func (tn TypeName) IsMergeableManageableResources() bool {
	_, ok := tn.ManageableResource().(resources.MergeableManageableResources)
	return ok
}

func NewTypeName(tpe resources.Type, name resources.Name) (TypeName, error) {
	return NewTypeNameFromStr(fmt.Sprintf("%s[%s]", tpe, name))
}

func MustNewTypeName(tpe resources.Type, name resources.Name) TypeName {
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

// Holds a resource definition, used for marshalling.
type ResourceDef struct {
	TypeName TypeName        `yaml:"resource"`
	State    resources.State `yaml:"state"`
	Destroy  bool            `yaml:"destroy"`
}

type resourceUnmarshalSchema struct {
	TypeName  TypeName  `yaml:"resource"`
	StateNode yaml.Node `yaml:"state"`
	Destroy   bool      `yaml:"destroy"`
}

func (r *ResourceDef) UnmarshalYAML(node *yaml.Node) error {
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

	stateInstance, ok := resources.ManageableResourcesStateMap[tpe]
	if !ok {
		panic(fmt.Errorf("Type %s missing from ManageableResourcesStateMap", tpe))
	}
	var state resources.State
	if unmarshalSchema.Destroy {
		if unmarshalSchema.StateNode.Content != nil {
			return fmt.Errorf("line %d: can not set state when destroy is set", node.Line)
		}
	} else {
		stateInstance := reflect.New(reflect.TypeOf(stateInstance)).Interface().(resources.State)
		unmarshalSchema.StateNode.KnownFields(true)
		err := unmarshalSchema.StateNode.Decode(stateInstance)
		if err != nil {
			return fmt.Errorf("line %d: %w", unmarshalSchema.StateNode.Line, err)
		}
		state = reflect.ValueOf(stateInstance).Elem().Interface().(resources.State)
	}

	*r = NewResourceDef(
		unmarshalSchema.TypeName,
		state,
		unmarshalSchema.Destroy,
	)
	return nil
}

func (r ResourceDef) MarshalYAML() (interface{}, error) {
	if r.Destroy {
		r.State = nil
	}
	type resourceDefAlias ResourceDef
	node := yaml.Node{}
	err := node.Encode(resourceDefAlias(r))
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (r ResourceDef) MustType() resources.Type {
	return r.TypeName.Type()
}

func (r ResourceDef) MustName() resources.Name {
	return r.TypeName.Name()
}

func (r ResourceDef) String() string {
	return string(r.TypeName)
}

func (r ResourceDef) ManageableResource() resources.ManageableResource {
	return r.TypeName.ManageableResource()
}

// Refreshable returns whether the resource is refreshable or not.
func (r ResourceDef) Refreshable() bool {
	_, ok := r.ManageableResource().(resources.RefreshableManageableResource)
	return ok
}

// MustIndividuallyManageableResource returns IndividuallyManageableResource from ManageableResource or
// panics if it isn't of the required type.
func (r ResourceDef) MustIndividuallyManageableResource() resources.IndividuallyManageableResource {
	individuallyManageableResource, ok := r.ManageableResource().(resources.IndividuallyManageableResource)
	if !ok {
		panic(fmt.Errorf("%s is not IndividuallyManageableResource", r))
	}
	return individuallyManageableResource
}

// IsMergeableManageableResources returns true only if ManageableResource is of type MergeableManageableResources.
func (r ResourceDef) IsMergeableManageableResources() bool {
	_, ok := r.ManageableResource().(resources.MergeableManageableResources)
	return ok
}

// MustMergeableManageableResources returns MergeableManageableResources from ManageableResource or
// panics if it isn't of the required type.
func (r ResourceDef) MustMergeableManageableResources() resources.MergeableManageableResources {
	mergeableManageableResources, ok := r.ManageableResource().(resources.MergeableManageableResources)
	if !ok {
		panic(fmt.Errorf("%s is not MergeableManageableResources", r))
	}
	return mergeableManageableResources
}

func NewResourceDef(typeName TypeName, state resources.State, destroy bool) ResourceDef {
	return ResourceDef{
		TypeName: typeName,
		State:    state,
		Destroy:  destroy,
	}
}
