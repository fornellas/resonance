package blueprint

import (
	"context"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

// Step represent a SingleResource or list of GroupResource, which must be applied in a single step.
type Step struct {
	singleResource resourcesPkg.SingleResource
	groupResource  resourcesPkg.GroupResource
	groupResources resourcesPkg.Resources
	requiredBy     []*Step
}

func NewSingleResourceStep(singleResource resourcesPkg.SingleResource) *Step {
	return &Step{
		singleResource: singleResource,
	}
}

func NewGroupResourceStep(groupResource resourcesPkg.GroupResource) *Step {
	return &Step{
		groupResource: groupResource,
	}
}

func (s *Step) String() string {
	if s.singleResource != nil {
		return fmt.Sprintf(
			"%s: %s",
			resourcesPkg.GetResourceTypeName(s.singleResource), resourcesPkg.GetResourceId(s.singleResource),
		)
	}

	if s.groupResource != nil {
		return fmt.Sprintf(
			"%s: %s",
			resourcesPkg.GetGroupResourceTypeName(s.groupResource), s.groupResources.Ids(),
		)
	}

	panic("bug: invalid state")
}

func (s *Step) AppendGroupResource(resource resourcesPkg.Resource) {
	if s.groupResource == nil {
		panic("bug: can't add Resource to non GroupResource Step")
	}
	s.groupResources = append(s.groupResources, resource)
}

func (s *Step) appendRequiredByStep(step *Step) {
	s.requiredBy = append(s.requiredBy, step)
}

func (s *Step) resolve(ctx context.Context, hst host.Host) error {
	if s.singleResource != nil {
		err := s.singleResource.Resolve(ctx, hst)
		if err != nil {
			return err
		}
		if err := resourcesPkg.ValidateResource(s.singleResource); err != nil {
			panic(fmt.Sprintf("bug: Resource Validate() failed after Resoslve(): %s", err.Error()))
		}
		return nil
	}

	if s.groupResource != nil {
		err := s.groupResource.Resolve(ctx, hst, s.groupResources)
		if err != nil {
			return err
		}
		if err := s.groupResources.Validate(); err != nil {
			panic(fmt.Sprintf("bug: Resource Validate() failed after Resoslve(): %s", err.Error()))
		}
		return nil
	}

	panic("bug: invalid state")
}

func (s *Step) load(ctx context.Context, hst host.Host) (*Step, error) {
	ns := *s
	if s.singleResource != nil {
		resosurce := resourcesPkg.NewResourceWithSameId(s.singleResource)

		var ok bool
		ns.singleResource, ok = resosurce.(resourcesPkg.SingleResource)
		if !ok {
			panic("bug: Resource is not SingleResource")
		}

		if err := ns.singleResource.Load(ctx, hst); err != nil {
			return nil, err
		}

		if err := resourcesPkg.ValidateResource(ns.singleResource); err != nil {
			panic(fmt.Sprintf("bug: Validate() after Load() failed: %s", err.Error()))
		}

		return &ns, nil
	}

	if s.groupResource != nil {
		ns.groupResource = s.groupResource
		ns.groupResources = resourcesPkg.NewResourcesWithSameIds(s.groupResources)

		if err := ns.groupResource.Load(ctx, hst, ns.groupResources); err != nil {
			return nil, err
		}

		if err := ns.groupResources.Validate(); err != nil {
			panic(fmt.Sprintf("bug: Validate() after Load() failed: %s", err.Error()))
		}

		return &ns, nil
	}

	panic("bug: invalid state")
}

func (s *Step) MarshalYAML() (interface{}, error) {
	type MarshalSchema struct {
		SingleResourceType string                      `yaml:"single_resource_type,omitempty"`
		SingleResource     resourcesPkg.SingleResource `yaml:"single_resource,omitempty"`
		GroupResourceType  string                      `yaml:"group_resource_type,omitempty"`
		GroupResourcesType string                      `yaml:"group_resources_type,omitempty"`
		GroupResources     resourcesPkg.Resources      `yaml:"group_resources,omitempty"`
	}

	var singleResourceType string
	if s.singleResource != nil {
		singleResourceType = reflect.TypeOf(s.singleResource).Elem().Name()
	}

	var groupResourceType string
	var groupResourcesType string
	if s.groupResource != nil {
		groupResourceType = reflect.TypeOf(s.groupResource).Elem().Name()
		groupResourcesType = reflect.TypeOf(s.groupResources[0]).Elem().Name()
	}

	return &MarshalSchema{
		SingleResourceType: singleResourceType,
		SingleResource:     s.singleResource,
		GroupResourceType:  groupResourceType,
		GroupResourcesType: groupResourcesType,
		GroupResources:     s.groupResources,
	}, nil
}

func (s *Step) UnmarshalYAML(node *yaml.Node) error {
	type UnmarshalSchema struct {
		SingleResourceType *string                `yaml:"single_resource_type"`
		SingleResourceNode yaml.Node              `yaml:"single_resource"`
		GroupResourceType  *string                `yaml:"group_resource_type"`
		GroupResourcesType *string                `yaml:"group_resources_type"`
		GroupResources     resourcesPkg.Resources `yaml:"group_resources"`
	}

	step := &UnmarshalSchema{}

	node.KnownFields(true)
	err := node.Decode(step)
	if err != nil {
		return fmt.Errorf("line %d: %s", node.Line, err.Error())
	}

	if step.SingleResourceType != nil {
		s.singleResource = resourcesPkg.GetSingleResourceByTypeName(*step.SingleResourceType)
		if s.singleResource == nil {
			return fmt.Errorf("line %d: invalid single resource type: %#v ", node.Line, *step.SingleResourceType)
		}
		step.SingleResourceNode.KnownFields(true)
		err := step.SingleResourceNode.Decode(s.singleResource)
		if err != nil {
			return fmt.Errorf("line %d: %s", node.Line, err.Error())
		}
	}

	if step.GroupResourceType != nil {
		s.groupResource = resourcesPkg.GetGroupResourceByTypeName(*step.GroupResourcesType)
		if s.groupResource == nil {
			return fmt.Errorf("line %d: invalid group resource type: %#v ", node.Line, *step.GroupResourceType)
		}
		s.groupResources = step.GroupResources
	}

	if s.singleResource != nil && s.groupResource != nil {
		panic("bug: YAML contents does not reflect schema: it defines both SingleResource and GroupResource")
	}

	if s.singleResource == nil && s.groupResource == nil {
		panic("bug: YAML contents does not reflect schema: it does not define either SingleResource or GroupResource")
	}

	return nil
}

// Returns all resources from step, ordered.
func (s *Step) Resources() resourcesPkg.Resources {
	resources := resourcesPkg.Resources{}

	if s.singleResource != nil {
		resources = append(resources, s.singleResource.(resourcesPkg.Resource))
	}

	resources = append(resources, s.groupResources...)

	return resources
}
