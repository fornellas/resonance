package blueprint

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

// Step represent a SingleResource or list of GroupResource, which must be applied in a single step.
type Step struct {
	singleResource resourcesPkg.SingleResource
	groupResource  resourcesPkg.GroupResource
	groupResources resourcesPkg.Resources
	requiredBy     Steps
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

func (s *Step) IsSingleResource() bool {
	return s.singleResource != nil
}

func (s *Step) IsGroupResource() bool {
	return s.groupResource != nil
}

func (s *Step) MustGroupResource() resourcesPkg.GroupResource {
	if s.groupResource == nil {
		panic("bug: not a GroupResource")
	}
	return s.groupResource
}

func (s *Step) Type() string {
	if s.singleResource != nil {
		return resourcesPkg.GetResourceTypeName(s.singleResource)
	}

	if s.groupResource != nil {
		return resourcesPkg.GetGroupResourceTypeName(s.groupResource)
	}

	panic("bug: invalid state")
}

func (s *Step) String() string {
	if s.singleResource != nil {
		return resourcesPkg.GetResourceTypeId(s.singleResource)
	}

	if s.groupResource != nil {
		return fmt.Sprintf(
			"%s:%s",
			resourcesPkg.GetGroupResourceTypeName(s.groupResource), s.groupResources.Ids(),
		)
	}

	panic("bug: invalid state")
}

func (s *Step) DetailedString() string {
	var buff bytes.Buffer

	fmt.Fprintf(&buff, "%s:\n", s.Type())

	for _, resource := range s.Resources() {
		lines := strings.Split(
			resourcesPkg.GetResourceYaml(resource),
			"\n",
		)
		if s.singleResource != nil {
			fmt.Fprintf(
				&buff, "  %s",
				strings.Join(lines, "\n  "),
			)
		}
		if s.groupResource != nil {
			fmt.Fprintf(
				&buff, "  - %s\n",
				strings.Join(lines, "\n    "),
			)
		}
	}

	return strings.TrimSuffix(buff.String(), "\n")
}

func (s *Step) AppendGroupResource(resource resourcesPkg.Resource) {
	if s.groupResource == nil {
		panic("bug: can't add Resource to non GroupResource Step")
	}
	s.groupResources = append(s.groupResources, resource)
	slices.SortFunc(s.groupResources, func(a, b resourcesPkg.Resource) int {
		return strings.Compare(
			resourcesPkg.GetResourceId(a), resourcesPkg.GetResourceId(b),
		)
	})
}

func (s *Step) appendRequiredByStep(step *Step) {
	s.requiredBy = append(s.requiredBy, step)
}

// Resolve the state with information that may be required from the host for all Resources.
func (s *Step) Resolve(ctx context.Context, hst host.Host) error {
	ctx, _ = log.MustContextLoggerSection(ctx, s.String())

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

// Load returns a copy of the Step, with all resource states loaded from given Host.
func (s *Step) Load(ctx context.Context, hst host.Host) (*Step, error) {
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

type Steps []*Step

func NewSteps(resources resourcesPkg.Resources) (Steps, error) {
	steps := Steps{}

	typeToStepMap := map[string]*Step{}

	requiredSteps := Steps{}
	pastRequiredSteps := map[*Step]bool{}

	for _, resource := range resources {
		var step *Step = nil

		typeName := reflect.TypeOf(resource).Elem().Name()

		if resourcesPkg.IsGroupResource(typeName) {
			var ok bool
			step, ok = typeToStepMap[typeName]
			if !ok {
				groupResource := resourcesPkg.GetGroupResourceByTypeName(typeName)
				if groupResource == nil {
					panic("bug: invalid resource type")
				}
				step = NewGroupResourceStep(groupResource)
				typeToStepMap[typeName] = step
				steps = append(steps, step)
			}
			step.AppendGroupResource(resource)
		} else {
			singleResource, ok := resource.(resourcesPkg.SingleResource)
			if !ok {
				panic(fmt.Sprintf("bug: Resource is not SingleResource: %#v", resource))
			}
			step = NewSingleResourceStep(singleResource)
			steps = append(steps, step)
		}

		var extraRequiredStep *Step = nil
		for _, requiredStep := range requiredSteps {
			if _, ok := pastRequiredSteps[step]; !ok {
				if requiredStep != step {
					requiredStep.appendRequiredByStep(step)
					pastRequiredSteps[requiredStep] = true
				}
			} else {
				extraRequiredStep = requiredStep
			}
		}

		requiredSteps = Steps{step}
		if extraRequiredStep != nil {
			requiredSteps = append(requiredSteps, extraRequiredStep)
		}
	}

	steps, err := topologicalSortSteps(steps)
	if err != nil {
		return nil, err
	}

	return steps, nil
}

func topologicalSortSteps(steps Steps) (Steps, error) {
	dependantCount := map[*Step]int{}
	for _, step := range steps {
		if _, ok := dependantCount[step]; !ok {
			dependantCount[step] = 0
		}
		for _, requiredStep := range step.requiredBy {
			dependantCount[requiredStep]++
		}
	}

	noDependantsSteps := Steps{}
	for _, step := range steps {
		if dependantCount[step] == 0 {
			noDependantsSteps = append(noDependantsSteps, step)
		}
	}

	sortedSteps := Steps{}
	for len(noDependantsSteps) > 0 {
		step := noDependantsSteps[0]
		noDependantsSteps = noDependantsSteps[1:]
		sortedSteps = append(sortedSteps, step)
		for _, dependantStep := range step.requiredBy {
			dependantCount[dependantStep]--
			if dependantCount[dependantStep] == 0 {
				noDependantsSteps = append(noDependantsSteps, dependantStep)
			}
		}
	}

	if len(sortedSteps) != len(steps) {
		return nil, errors.New("unable to topological sort, cycle detected")
	}

	return sortedSteps, nil
}
