package blueprint

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/barkimedes/go-deepcopy"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

// Step represent a SingleResource or list of GroupResource, which must be applied in a single step.
type Step struct {
	singleResource resourcesPkg.SingleResource
	groupResource  resourcesPkg.GroupResource
	groupResources resourcesPkg.Resources
	requiredBy     []*Step
}

func newSingleResourceStep(singleResource resourcesPkg.SingleResource) *Step {
	return &Step{
		singleResource: singleResource,
	}
}

func newGroupResourceStep(groupResource resourcesPkg.GroupResource) *Step {
	return &Step{
		groupResource: groupResource,
	}
}

func (s *Step) String() string {
	if s.singleResource != nil {
		return fmt.Sprintf(
			"%s: %s",
			reflect.TypeOf(s.singleResource).Elem().Name(), resourcesPkg.GetResourceId(s.singleResource),
		)
	}

	if s.groupResource != nil {
		return fmt.Sprintf(
			"%s: %s",
			reflect.TypeOf(s.groupResource).Elem().Name(), s.groupResources.Ids(),
		)
	}

	panic("bug: invalid state")
}

func (s *Step) appendGroupResource(resource resourcesPkg.Resource) {
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
		if err := s.singleResource.Validate(); err != nil {
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

func (s *Step) load(ctx context.Context, hst host.Host) error {
	if s.singleResource != nil {
		if err := s.singleResource.Load(ctx, hst); err != nil {
			return err
		}
		if err := s.singleResource.Validate(); err != nil {
			panic(fmt.Sprintf("bug: Validate() after Load() failed: %s", err.Error()))
		}
		return nil
	}

	if s.groupResource != nil {
		if err := s.groupResource.Load(ctx, hst, s.groupResources); err != nil {
			return err
		}
		if err := s.groupResources.Validate(); err != nil {
			panic(fmt.Sprintf("bug: Validate() after Load() failed: %s", err.Error()))
		}
		return nil
	}

	panic("bug: invalid state")
}

func (s *Step) check(ctx context.Context, hst host.Host) (bool, error) {
	if s.singleResource != nil {
		currentSingleResourceInterface := deepcopy.MustAnything(s.singleResource)
		currentSingleResource := currentSingleResourceInterface.(resourcesPkg.SingleResource)
		err := currentSingleResource.Load(ctx, hst)
		if err != nil {
			return false, err
		}
		if !reflect.DeepEqual(s.singleResource, currentSingleResource) {
			return false, nil
		}
		return true, nil
	}

	if s.groupResource != nil {
		currentGroupResourcesInterface := deepcopy.MustAnything(s.groupResources)
		currentGroupResources := currentGroupResourcesInterface.(resourcesPkg.Resources)
		err := s.groupResource.Load(ctx, hst, currentGroupResources)
		if err != nil {
			return false, err
		}
		if !reflect.DeepEqual(s.groupResources, currentGroupResources) {
			return false, nil
		}
		return true, nil
	}

	panic("bug: invalid state")
}

func (s *Step) MarshalYAML() (interface{}, error) {
	type marshalSchema struct {
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

	return &marshalSchema{
		SingleResourceType: singleResourceType,
		SingleResource:     s.singleResource,
		GroupResourceType:  groupResourceType,
		GroupResourcesType: groupResourcesType,
		GroupResources:     s.groupResources,
	}, nil
}

// Blueprint holds a full desired state for a host.
type Blueprint []*Step

func newBlueprintFromRessources(resources resourcesPkg.Resources) Blueprint {
	blueprint := Blueprint{}

	typeToStepMap := map[string]*Step{}

	requiredSteps := Blueprint{}
	pastRequiredSteps := map[*Step]bool{}

	for _, resource := range resources {
		var step *Step = nil

		typeName := reflect.TypeOf(resource).Elem().Name()

		if resourcesPkg.IsGroupResource(typeName) {
			var ok bool
			step, ok = typeToStepMap[typeName]
			if !ok {
				step = newGroupResourceStep(
					resourcesPkg.GetGroupResourceByTypeName(typeName),
				)
				typeToStepMap[typeName] = step
				blueprint = append(blueprint, step)
			}
			step.appendGroupResource(resource)
		} else {
			singleResource, ok := resource.(resourcesPkg.SingleResource)
			if !ok {
				panic("bug: Resource is not SingleResource")
			}
			step = newSingleResourceStep(singleResource)
			blueprint = append(blueprint, step)
		}

		var extraRequiredStep *Step = nil
		for _, requiredStep := range requiredSteps {
			if _, ok := pastRequiredSteps[step]; !ok {
				requiredStep.appendRequiredByStep(step)
				pastRequiredSteps[requiredStep] = true
			} else {
				extraRequiredStep = requiredStep
			}
		}

		requiredSteps = Blueprint{step}
		if extraRequiredStep != nil {
			requiredSteps = append(requiredSteps, extraRequiredStep)
		}
	}

	return blueprint
}

// NewBlueprintFromResources creates a new Blueprint from given Resources, merging GroupResource
// of the same type in the same step, while respecting the declared order.
func NewBlueprintFromResources(ctx context.Context, resources resourcesPkg.Resources, hst host.Host) (Blueprint, error) {
	logger := log.MustLogger(ctx).WithGroup("blueprint")
	ctx = log.WithLogger(ctx, logger)
	logger.Info("⚙️ Computing blueprint")

	blueprint := newBlueprintFromRessources(resources)

	blueprint, err := blueprint.topologicalSsort()
	if err != nil {
		return nil, err
	}

	if err := blueprint.resolve(ctx, hst); err != nil {
		return nil, err
	}

	return blueprint, nil
}

// NewPlanBlueprint calculates a Blueprint based on the lastBlueprint, representing a committed host
// state, and targetBlueprint, representing the delined host state.
func NewPlanBlueprint(ctx context.Context, lastBlueprint Blueprint, targetBlueprint Blueprint) (Blueprint, error) {
	// for resource
	//   if in lastBlueprint and NOT in targetBlueprint
	//     destroy
	//   if in lastBlueprint AND in targetBlueprint
	//     if equal
	//       do nothing
	//     else
	//       diff
	//       apply
	//   if NOT in lastBlueprint and in targetBlueprint
	//     apply
	panic("TODO")
}

func (b Blueprint) String() string {
	var buff bytes.Buffer
	for i, step := range b {
		i++
		fmt.Fprintf(&buff, "%d. %s\n", i, step)
	}
	return buff.String()
}

// Copy returns a deep copy.
func (b Blueprint) Copy() Blueprint {
	return deepcopy.MustAnything(b).(Blueprint)
}

// Load the full current state of all resourcess from given Host.
func (b Blueprint) Load(ctx context.Context, hst host.Host) error {
	for _, step := range b {
		if err := step.load(ctx, hst); err != nil {
			return err
		}
	}
	return nil
}

// Check whether given Host state matches the Blueprint.
func (b Blueprint) Check(ctx context.Context, hst host.Host) (bool, error) {
	logger := log.MustLogger(ctx)

	for _, step := range b {
		ok, err := step.check(ctx, hst)
		if err != nil {
			return false, err
		}
		if !ok {
			logger.Warn("host state changed", "step", step.String())
		}
	}
	return true, nil
}

func (b Blueprint) topologicalSsort() (Blueprint, error) {
	dependantCount := map[*Step]int{}
	for _, step := range b {
		if _, ok := dependantCount[step]; !ok {
			dependantCount[step] = 0
		}
		for _, requiredStep := range step.requiredBy {
			dependantCount[requiredStep]++
		}
	}

	noDependantsSteps := Blueprint{}
	for _, step := range b {
		if dependantCount[step] == 0 {
			noDependantsSteps = append(noDependantsSteps, step)
		}
	}

	sortedSteps := Blueprint{}
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

	if len(sortedSteps) != len(b) {
		return nil, errors.New("unable to topological sort, cycle detected")
	}

	return sortedSteps, nil
}

func (b Blueprint) resolve(ctx context.Context, hst host.Host) error {
	for _, step := range b {
		if err := step.resolve(ctx, hst); err != nil {
			return err
		}
	}
	return nil
}
