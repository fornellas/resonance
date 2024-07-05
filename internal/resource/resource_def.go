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

// Resource returns an instance for the resource type.
func (tn TypeName) Resource() resources.Resource {
	tpe, _, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return tpe.Resource()
}

// IsIndividualResource returns true if Resource() is of type IndividualResource.
func (tn TypeName) IsIndividualResource() bool {
	_, ok := tn.Resource().(resources.IndividualResource)
	return ok
}

// MustIndividualResource returns IndividualResource from Resource or
// panics if it isn't of the required type.
func (tn TypeName) MustIndividualResource() resources.IndividualResource {
	individualResource, ok := tn.Resource().(resources.IndividualResource)
	if !ok {
		panic(fmt.Errorf("%s is not IndividualResource", tn))
	}
	return individualResource
}

// IsMergeableResources returns true only if Resource() is of type MergeableResources.
func (tn TypeName) IsMergeableResources() bool {
	_, ok := tn.Resource().(resources.MergeableResources)
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

	resource := unmarshalSchema.TypeName.Resource()
	tpe := unmarshalSchema.TypeName.Type()
	name := unmarshalSchema.TypeName.Name()
	if err := resource.ValidateName(name); err != nil {
		return fmt.Errorf("line %d: %w", node.Line, err)
	}

	stateInstance, ok := resources.ResourcesStateMap[tpe]
	if !ok {
		panic(fmt.Errorf("Type %s missing from ResourcesStateMap", tpe))
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

func (r ResourceDef) Resource() resources.Resource {
	return r.TypeName.Resource()
}

// Refreshable returns whether the resource is refreshable or not.
func (r ResourceDef) Refreshable() bool {
	_, ok := r.Resource().(resources.RefreshableResource)
	return ok
}

// MustIndividualResource returns IndividualResource from Resource or
// panics if it isn't of the required type.
func (r ResourceDef) MustIndividualResource() resources.IndividualResource {
	individualResource, ok := r.Resource().(resources.IndividualResource)
	if !ok {
		panic(fmt.Errorf("%s is not IndividualResource", r))
	}
	return individualResource
}

// IsMergeableResources returns true only if Resource is of type MergeableResources.
func (r ResourceDef) IsMergeableResources() bool {
	_, ok := r.Resource().(resources.MergeableResources)
	return ok
}

// MustMergeableResources returns MergeableResources from Resource or
// panics if it isn't of the required type.
func (r ResourceDef) MustMergeableResources() resources.MergeableResources {
	mergeableResources, ok := r.Resource().(resources.MergeableResources)
	if !ok {
		panic(fmt.Errorf("%s is not MergeableResources", r))
	}
	return mergeableResources
}

func NewResourceDef(typeName TypeName, state resources.State, destroy bool) ResourceDef {
	return ResourceDef{
		TypeName: typeName,
		State:    state,
		Destroy:  destroy,
	}
}
