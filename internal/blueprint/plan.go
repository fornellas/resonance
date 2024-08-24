package blueprint

import (
	"context"
	"fmt"
	"reflect"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/internal/diff"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

// ResourceAction details the exact action for a single Resource within an
// Action.
type ResourceAction struct {
	Emoji          string
	LastResource   resourcesPkg.Resource
	TargetResource resourcesPkg.Resource
}

func (r *ResourceAction) String() string {
	return fmt.Sprintf("%s %s", r.Emoji, resourcesPkg.GetResourceId(r.TargetResource))
}

func (r *ResourceAction) Diff() diff.Chunks {
	return diff.DiffAsYaml(r.LastResource, r.TargetResource)
}

// Action represents a single transactional step of a Plan.
type Action struct {
	Step            *Step
	ResourceActions []ResourceAction
}

func NewAction(step *Step, lastResourceMap resourcesPkg.ResourceMap) *Action {
	action := &Action{
		Step:            step,
		ResourceActions: []ResourceAction{},
	}

	for _, targetResource := range step.Resources() {
		lastResource := lastResourceMap.GetResourceWithSameTypeId(targetResource)
		if lastResource == nil {
			action.addResourceAction("🛠️", nil, targetResource)
		} else {
			// FIXME reflect.DeepEqual does not work for APTPackage:
			// - When loaded, it always come with the version.
			// - If new state is the same package, but does not specify a version
			//    - It differs
			//    - It should not differ, as the new state requires no change to be applied
			if reflect.DeepEqual(lastResource, targetResource) {
				action.addResourceAction("✅", lastResource, targetResource)
			} else {
				if resourcesPkg.GetResourceRemove(lastResource) {
					if resourcesPkg.GetResourceRemove(targetResource) {
						panic("bug: both last and target resources are marked to be removed but differ")
					} else {
						action.addResourceAction("🛠️", nil, targetResource)
					}
				} else {
					// FIXME this must restore the original state when the resource is no longer managed
					if resourcesPkg.GetResourceRemove(targetResource) {
						action.addResourceAction("🗑️", lastResource, targetResource)
					} else {
						action.addResourceAction("🔄", lastResource, targetResource)
					}
				}
			}
		}
	}

	if len(action.ResourceActions) == 0 {
		panic("bug: Action has no ResourceActions")
	}

	return action
}

func (a *Action) String() string {
	return a.Step.String()
}

func (a *Action) addResourceAction(
	emoji string, lastResource, targetResource resourcesPkg.Resource,
) {
	a.ResourceActions = append(
		a.ResourceActions,
		ResourceAction{
			Emoji:          emoji,
			LastResource:   lastResource,
			TargetResource: targetResource,
		},
	)
}

// Plan is used to calculate the required list of Action required to apply a Blueprint, as a
// function of the previously committed Blueprint.
// This enables detailing exactly what happens to each Resource: nothing, create, update or remove.
type Plan struct {
	Blueprint         *Blueprint
	BeforeResourceMap resourcesPkg.ResourceMap
}

// NewPlan crafts a new plan as a function of the target state, last state and the original state.
func NewPlan(
	ctx context.Context, hst host.Host,
	targetBlueprint, lastBlueprint *Blueprint,
	loadOriginalResource func(ctx context.Context, resource resourcesPkg.Resource) (resourcesPkg.Resource, error),
) (*Plan, error) {
	targetResources := targetBlueprint.Resources()
	planResources := make(resourcesPkg.Resources, len(targetResources))
	beforeResources := make(resourcesPkg.Resources, len(targetResources))

	// apply all target resources
	for i, targetResource := range targetResources {
		planResources[i] = targetResource
		lastResource := lastBlueprint.GetResourceWithSameTypeId(targetResource)
		if lastResource != nil {
			beforeResources[i] = lastResource
		} else {
			originalResource, err := loadOriginalResource(ctx, targetResource)
			if err != nil {
				return nil, err
			}
			beforeResources[i] = originalResource
		}
	}

	// if resource is on last, but not in target, restore it to original
	for _, lastResource := range lastBlueprint.Resources() {
		if targetBlueprint.GetResourceWithSameTypeId(lastResource) == nil {
			originalResource, err := loadOriginalResource(ctx, lastResource)
			if err != nil {
				return nil, err
			}
			planResources = append(planResources, originalResource)
			beforeResources = append(beforeResources, lastResource)
		}
	}

	planBlueprint, err := NewBlueprintFromResources(ctx, planResources)
	if err != nil {
		return nil, err
	}

	return &Plan{
		Blueprint:         planBlueprint,
		BeforeResourceMap: resourcesPkg.NewResourceMap(beforeResources),
	}, nil
}

// GetActions returns detailed actions required to apply the plan.
func (p *Plan) GetActions() []*Action {
	actions := make([]*Action, len(p.Blueprint.Steps))
	for i, step := range p.Blueprint.Steps {
		action := NewAction(step, p.BeforeResourceMap)
		actions[i] = action
	}
	return actions
}
