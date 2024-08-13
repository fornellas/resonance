package blueprint

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

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
				groupResource := resourcesPkg.GetGroupResourceByTypeName(typeName)
				if groupResource == nil {
					panic("bug: invalid resource type")
				}
				step = NewGroupResourceStep(groupResource)
				typeToStepMap[typeName] = step
				blueprint = append(blueprint, step)
			}
			step.AppendGroupResource(resource)
		} else {
			singleResource, ok := resource.(resourcesPkg.SingleResource)
			if !ok {
				panic(fmt.Sprintf("bug: Resource is not SingleResource: %#v", resource))
			}
			step = NewSingleResourceStep(singleResource)
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

// Load returns a copy of the Blueprint, with all resource states loaded from given Host.
func (b Blueprint) Load(ctx context.Context, hst host.Host) (Blueprint, error) {
	logger := log.MustLogger(ctx)
	logger.Info("🔍 Loading Blueprint from host")
	newBlueprint := make(Blueprint, len(b))
	for i, step := range b {
		newStep, err := step.load(ctx, hst)
		if err != nil {
			return nil, err
		}
		newBlueprint[i] = newStep
	}
	return newBlueprint, nil
}
