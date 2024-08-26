package plan

import (
	"context"

	"github.com/fornellas/resonance/host"
	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

type Plan struct {
}

func NewPlan(
	ctx context.Context, hst host.Host,
	targetBlueprint, lastBlueprint *blueprintPkg.Blueprint,
	loadOriginalResource func(ctx context.Context, resource resourcesPkg.Resource) (resourcesPkg.Resource, error),
) (*Plan, error) {
	targetResources := targetBlueprint.Resources()
	planResources := make(resourcesPkg.Resources, len(targetResources))
	beforeResources := make(resourcesPkg.Resources, len(targetResources))

	for i, targetResource := range targetResources {
		lastResource := lastBlueprint.GetResourceWithSameTypeId(targetResource)
		planResources[i] = targetResource
		if lastResource == nil {
			originalResource, err := loadOriginalResource(ctx, targetResource)
			if err != nil {
				return nil, err
			}
			beforeResources[i] = originalResource
		} else {
			beforeResources[i] = lastResource
		}
	}

	for _, lastResource := range lastBlueprint.Resources() {
		if targetBlueprint.GetResourceWithSameTypeId(lastResource) != nil {
			continue
		}
		originalResource, err := loadOriginalResource(ctx, lastResource)
		if err != nil {
			return nil, err
		}
		// FIXME prepending this has potential to create cycles
		// GroupResource should be together with the same time
		// SingleResource, should respect order of merged target & last blueprints
		planResources = append(resourcesPkg.Resources{originalResource}, planResources...)
		beforeResources = append(beforeResources, lastResource)
	}

	planBlueprint, err := blueprintPkg.NewBlueprintFromResources(ctx, planResources)
	if err != nil {
		return nil, err
	}

	beforeResourceMap := resourcesPkg.NewResourceMap(beforeResources)

	plan := &Plan{}
	for _, step := range planBlueprint.Steps {
		// TODO new action
		for _, planResource := range step.Resources() {
			beforeResource := beforeResourceMap.GetResourceWithSameTypeId(planResource)
			if beforeResource == nil {
				panic("bug: before resource not found")
			}
			if resourcesPkg.Satisfies(beforeResource, planResource) {
				// TODO append to action ✅ no action
			} else {
				if resourcesPkg.GetResourceRemove(beforeResource) {
					// TODO append to action 🛠️ create, diff(nil, planResource)
				} else {
					if resourcesPkg.GetResourceRemove(planResource) {
						// TODO append to action 🗑️ remove, diff(beforeResource, planResource)
					} else {
						// TODO append to action 🔄 update, diff(beforeResource, planResource)
					}
				}
			}
		}
		// TODO add action to plan
	}
	return plan, nil
}
