package blueprint

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/fornellas/resonance/host"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

// Blueprint holds a full desired state for a host.
type Blueprint struct {
	Steps       []*Step
	resourceMap resourcesPkg.ResourceMap
}

func newBlueprintFromRessources(resources resourcesPkg.Resources) *Blueprint {
	blueprint := &Blueprint{}

	typeToStepMap := map[string]*Step{}

	requiredSteps := []*Step{}
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
				blueprint.Steps = append(blueprint.Steps, step)
			}
			step.AppendGroupResource(resource)
		} else {
			singleResource, ok := resource.(resourcesPkg.SingleResource)
			if !ok {
				panic(fmt.Sprintf("bug: Resource is not SingleResource: %#v", resource))
			}
			step = NewSingleResourceStep(singleResource)
			blueprint.Steps = append(blueprint.Steps, step)
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

		requiredSteps = []*Step{step}
		if extraRequiredStep != nil {
			requiredSteps = append(requiredSteps, extraRequiredStep)
		}
	}

	return blueprint
}

// NewBlueprintFromResources creates a new Blueprint from given Resources, merging GroupResource
// of the same type in the same step, while respecting the declared order.
func NewBlueprintFromResources(ctx context.Context, resources resourcesPkg.Resources) (*Blueprint, error) {
	blueprint := newBlueprintFromRessources(resources)

	blueprint, err := blueprint.topologicalSort()
	if err != nil {
		return nil, err
	}

	blueprint.resourceMap = resourcesPkg.NewResourceMap(blueprint.Resources())

	return blueprint, nil
}

func (b *Blueprint) String() string {
	var buff bytes.Buffer
	for _, step := range b.Steps {
		fmt.Fprintf(&buff, "%s\n", step)
	}
	return buff.String()
}

func (b *Blueprint) topologicalSort() (*Blueprint, error) {
	dependantCount := map[*Step]int{}
	for _, step := range b.Steps {
		if _, ok := dependantCount[step]; !ok {
			dependantCount[step] = 0
		}
		for _, requiredStep := range step.requiredBy {
			dependantCount[requiredStep]++
		}
	}

	noDependantsSteps := []*Step{}
	for _, step := range b.Steps {
		if dependantCount[step] == 0 {
			noDependantsSteps = append(noDependantsSteps, step)
		}
	}

	sortedSteps := []*Step{}
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

	if len(sortedSteps) != len(b.Steps) {
		return nil, errors.New("unable to topological sort, cycle detected")
	}

	return &Blueprint{
		Steps: sortedSteps,
	}, nil
}

// Resolve the state with information that may be required from the host for all Resources.
func (b *Blueprint) Resolve(ctx context.Context, hst host.Host) error {
	for _, step := range b.Steps {
		if err := step.Resolve(ctx, hst); err != nil {
			return err
		}
	}
	return nil
}

// Load returns a copy of the Blueprint, with all resource states loaded from given Host.
func (b *Blueprint) Load(ctx context.Context, hst host.Host) (*Blueprint, error) {
	newBlueprint := &Blueprint{
		Steps: make([]*Step, len(b.Steps)),
	}
	for i, step := range b.Steps {
		newStep, err := step.Load(ctx, hst)
		if err != nil {
			return nil, err
		}
		newBlueprint.Steps[i] = newStep
	}
	return newBlueprint, nil
}

// Returns all resources from all steps, ordered.
func (b *Blueprint) Resources() resourcesPkg.Resources {
	resources := resourcesPkg.Resources{}
	if b == nil {
		return resources
	}
	for _, step := range b.Steps {
		resources = append(resources, step.Resources()...)
	}
	return resources
}

func (b *Blueprint) GetResourceWithSameTypeId(resource resourcesPkg.Resource) resourcesPkg.Resource {
	return b.resourceMap.GetResourceWithSameTypeId(resource)
}
