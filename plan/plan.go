package plan

import (
	"context"

	"fmt"

	blueprintPkg "github.com/fornellas/resonance/blueprint"
	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
	storePkg "github.com/fornellas/resonance/store"
)

// Plan holds all actions required to apply changes to a host.
// This enables evaluating all changes before they are applied.
type Plan []*Action

func compilePlan(
	ctx context.Context,
	targetBlueprint, lastBlueprint *blueprintPkg.Blueprint,
	loadOriginalResource func(ctx context.Context, resource resourcesPkg.Resource) (resourcesPkg.Resource, error),
) (Plan, error) {
	targetResources := targetBlueprint.Resources()
	planResources := make(resourcesPkg.Resources, len(targetResources))
	beforeResources := make(resourcesPkg.Resources, len(targetResources))

	// Add every target resource to plan
	for i, targetResource := range targetResources {
		planResources[i] = targetResource
		// Before resource is a function of last resource existing or not
		lastResource := lastBlueprint.GetResourceWithSameTypeId(targetResource)
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

	// each last resource that is not on target must be restored to original
	type ToOriginal struct {
		lastIdx  int
		resource resourcesPkg.Resource
	}
	toOriginalSlice := []ToOriginal{}
	lastResources := lastBlueprint.Resources()
	for i, lastResource := range lastResources {
		if targetBlueprint.HasResourceWithSameTypeId(lastResource) {
			continue
		}
		originalResource, err := loadOriginalResource(ctx, lastResource)
		if err != nil {
			return nil, err
		}
		toOriginalSlice = append(toOriginalSlice, ToOriginal{
			lastIdx:  i,
			resource: originalResource,
		})
		beforeResources = append(beforeResources, lastResource)
	}

	// merge original resources with plan resources, respecting the original order
	for _, toOriginal := range toOriginalSlice {
		idx := -1
		for i := toOriginal.lastIdx + 1; i < len(lastResources) && idx < 0; i++ {
			lastResourceTypeId := resourcesPkg.GetResourceTypeId(lastResources[i])
			for j, planResource := range planResources {
				planResourceTypeId := resourcesPkg.GetResourceTypeId(planResource)
				if planResourceTypeId == lastResourceTypeId {
					idx = j
					break
				}
			}
		}
		if idx < 0 {
			idx = len(planResources)
		}
		planResources = append(
			planResources[:idx],
			append(resourcesPkg.Resources{toOriginal.resource}, planResources[idx:]...)...,
		)
	}

	// Calculate plan actions
	planBlueprint, err := blueprintPkg.NewBlueprintFromResources(ctx, planResources)
	if err != nil {
		return nil, err
	}
	beforeResourceMap := resourcesPkg.NewResourceMap(beforeResources)
	plan := make(Plan, len(planBlueprint.Steps))
	for i, step := range planBlueprint.Steps {
		plan[i] = NewAction(step, beforeResourceMap)
	}
	return plan, nil
}

// createTargetBlueprint crafts a new target Blueprint from given unresolved target
// resources, against given host.
// The blueprint is saved at given store.
func createTargetBlueprint(
	ctx context.Context,
	host types.Host,
	targetResources resourcesPkg.Resources,
) (*blueprintPkg.Blueprint, error) {
	ctx, _ = log.WithGroup(ctx, "🎯 Crafting target Blueprint")

	var targetBlueprint *blueprintPkg.Blueprint
	{
		var err error
		targetBlueprint, err = blueprintPkg.NewBlueprintFromResources(ctx, targetResources)
		if err != nil {
			return nil, err
		}
		{
			ctx, _ := log.WithGroup(ctx, "⚙️ Resolving target Blueprint")
			if err := targetBlueprint.Resolve(ctx, host); err != nil {
				return nil, err
			}
		}
		{
			_, logger := log.WithGroup(ctx, "🧩 Crafted target Blueprint")
			for _, step := range targetBlueprint.Steps {
				resources := step.Resources()
				if len(resources) == 1 {
					logger.Info(resourcesPkg.GetResourceTypeName(resources[0]), "yaml", resourcesPkg.GetResourceYaml(resources[0]))
				} else {
					logger.Info(resourcesPkg.GetResourceTypeName(resources[0]), "yaml", resourcesPkg.GetResourcesYaml(resources))
				}
			}
		}
	}

	return targetBlueprint, nil
}

// saveOriginalResourcesState, for each resource from target Blueprint, loads the
// resource current state from given host, and saves it at store as the original
// state.
func saveOriginalResourcesState(
	ctx context.Context,
	host types.Host,
	store storePkg.Store,
	targetBlueprint *blueprintPkg.Blueprint,
) error {
	ctx, logger := log.WithGroup(ctx, "🌱 Storing original resource states")
	for _, step := range targetBlueprint.Steps {
		noOriginalResources := resourcesPkg.Resources{}
		for _, resource := range step.Resources() {
			resource = resourcesPkg.NewResourceWithSameId(resource)
			hasOriginal, err := store.HasOriginalResource(ctx, resource)
			if err != nil {
				return err
			}
			if !hasOriginal {
				noOriginalResources = append(noOriginalResources, resource)
			}
		}
		if len(noOriginalResources) == 0 {
			continue
		}
		var noOriginalStep *blueprintPkg.Step
		if step.IsSingleResource() {
			if len(noOriginalResources) != 1 {
				panic("bug: multiple single resource")
			}
			noOriginalStep = blueprintPkg.NewSingleResourceStep(noOriginalResources[0].(resourcesPkg.SingleResource))
		} else if step.IsGroupResource() {
			noOriginalStep = blueprintPkg.NewGroupResourceStep(step.MustGroupResource())
			for _, noOriginalResource := range noOriginalResources {
				noOriginalStep.AppendGroupResource(noOriginalResource)
			}
		} else {
			panic("bug: invalid step type")
		}
		originalStep, err := noOriginalStep.Load(ctx, host)
		if err != nil {
			return err
		}

		originalResources := originalStep.Resources()
		if len(originalResources) == 1 {
			logger.Info(resourcesPkg.GetResourceTypeName(originalResources[0]), "yaml", resourcesPkg.GetResourceYaml(originalResources[0]))
		} else {
			logger.Info(resourcesPkg.GetResourceTypeName(originalResources[0]), "yaml", resourcesPkg.GetResourcesYaml(originalResources))
		}

		for _, originalResource := range originalResources {
			if err := store.SaveOriginalResource(ctx, originalResource); err != nil {
				return err
			}
		}
	}
	return nil
}

// loadOrCreateAndSaveLastBlueprintWithValidation attempts to load last blueprint from store.
// If available, it validates whether the current host state matches that.
// If not available, it loads current host state for all resources from given target Blueprint
// and saves it.
func loadOrCreateAndSaveLastBlueprintWithValidation(
	ctx context.Context,
	host types.Host,
	store storePkg.Store,
	targetBlueprint *blueprintPkg.Blueprint,
) (*blueprintPkg.Blueprint, error) {
	ctx, logger := log.WithGroup(ctx, "↩️ Loading last Blueprint")
	lastBlueprint, err := store.LoadLastBlueprint(ctx)
	if err != nil {
		return nil, err
	}
	if lastBlueprint == nil {
		logger.Info("🔎 No last Blueprint, loading current state")
		var err error
		lastBlueprint, err = targetBlueprint.Load(ctx, host)
		if err != nil {
			return nil, err
		}

		{
			_, logger := log.WithGroup(ctx, "🧩 Loaded Blueprint")
			for _, step := range lastBlueprint.Steps {
				resources := step.Resources()
				if len(resources) == 1 {
					logger.Info(resourcesPkg.GetResourceTypeName(resources[0]), "yaml", resourcesPkg.GetResourceYaml(resources[0]))
				} else {
					logger.Info(resourcesPkg.GetResourceTypeName(resources[0]), "yaml", resourcesPkg.GetResourcesYaml(resources))
				}
			}
		}

		logger.Info("💾 Saving as last Blueprint")
		if err := store.SaveLastBlueprint(ctx, lastBlueprint); err != nil {
			return nil, err
		}
	} else {
		logger.Info("🔎 Validating previous host state")
		currentBlueprint, err := lastBlueprint.Load(ctx, host)
		if err != nil {
			return nil, err
		}

		chunks := currentBlueprint.Satisfies(lastBlueprint)
		if chunks.HaveChanges() {
			return nil, fmt.Errorf(
				"host state has changed:\n%s",
				chunks.String(),
			)
		}
	}
	return lastBlueprint, nil
}

// CraftPlan crafts a new plan as a function of given targetResources (desired state). It uses
// given store to persist the original state of all target resources (enabling them to be restored
// when not managed anymore). If a previous plan was previously applied, its state (last Blueprint)
// is loaded form store and validated (ensuring we're starting from a clean state). If no last
// Blueprint stored, then it loads the current state of all targetResources and save it as last
// Blueprint (enabling rollbacks).
func CraftPlan(
	ctx context.Context,
	host types.Host,
	store storePkg.Store,
	targetResources resourcesPkg.Resources,
) (Plan, *blueprintPkg.Blueprint, *blueprintPkg.Blueprint, error) {
	targetBlueprint, err := createTargetBlueprint(ctx, host, targetResources)
	if err != nil {
		return nil, nil, nil, err
	}

	err = saveOriginalResourcesState(ctx, host, store, targetBlueprint)
	if err != nil {
		return nil, nil, nil, err
	}

	lastBlueprint, err := loadOrCreateAndSaveLastBlueprintWithValidation(
		ctx, host, store, targetBlueprint,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	plan, err := compilePlan(
		ctx,
		targetBlueprint,
		lastBlueprint,
		store.LoadOriginalResource,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	return plan, targetBlueprint, lastBlueprint, nil
}

// Apply commits all required changes for plan to given Host.
func (p Plan) Apply(ctx context.Context, host types.Host) error {
	ctx, _ = log.WithGroup(ctx, "⚙️ Applying")

	for _, action := range p {
		if err := action.Apply(ctx, host); err != nil {
			return err
		}
	}

	return nil
}
