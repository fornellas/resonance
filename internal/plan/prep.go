package plan

import (
	"context"
	"fmt"

	"github.com/fornellas/resonance/internal/diff"

	hostPkg "github.com/fornellas/resonance/host"
	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
	storePkg "github.com/fornellas/resonance/internal/store"
	"github.com/fornellas/resonance/log"
	resouresPkg "github.com/fornellas/resonance/resources"
)

// CreateTargetBlueprint crafts a new target Blueprint from given unresolved target
// resources, against given host.
// The blueprint is saved at given store.
func CreateTargetBlueprint(
	ctx context.Context,
	host hostPkg.Host,
	targetResources resouresPkg.Resources,
) (*blueprintPkg.Blueprint, error) {
	ctx, _ = log.MustContextLoggerSection(ctx, "🎯 Crafting target Blueprint")

	var targetBlueprint *blueprintPkg.Blueprint
	{
		var err error
		targetBlueprint, err = blueprintPkg.NewBlueprintFromResources(ctx, targetResources)
		if err != nil {
			return nil, err
		}
		{
			ctx, _ := log.MustContextLoggerSection(ctx, "⚙️ Resolving target Blueprint")
			if err := targetBlueprint.Resolve(ctx, host); err != nil {
				return nil, err
			}
		}
		{
			_, logger := log.MustContextLoggerSection(ctx, "🧩 Crafted target Blueprint")
			for _, step := range targetBlueprint.Steps {
				resources := step.Resources()
				if len(resources) == 1 {
					logger.Info(resouresPkg.GetResourceTypeName(resources[0]), "yaml", resouresPkg.GetResourceYaml(resources[0]))
				} else {
					logger.Info(resouresPkg.GetResourceTypeName(resources[0]), "yaml", resouresPkg.GetResourcesYaml(resources))
				}
			}
		}
	}

	return targetBlueprint, nil
}

// SaveOriginalResourcesState, for each resource from target Blueprint, loads the
// resource current state from given host, and saves it at store as the original
// state.
func SaveOriginalResourcesState(
	ctx context.Context,
	host hostPkg.Host,
	store storePkg.Store,
	targetBlueprint *blueprintPkg.Blueprint,
) error {
	ctx, logger := log.MustContextLoggerSection(ctx, "🌱 Storing original resource states")
	for _, step := range targetBlueprint.Steps {
		noOriginalResources := resouresPkg.Resources{}
		for _, resource := range step.Resources() {
			resource = resouresPkg.NewResourceWithSameId(resource)
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
			noOriginalStep = blueprintPkg.NewSingleResourceStep(noOriginalResources[0].(resouresPkg.SingleResource))
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
			logger.Info(resouresPkg.GetResourceTypeName(originalResources[0]), "yaml", resouresPkg.GetResourceYaml(originalResources[0]))
		} else {
			logger.Info(resouresPkg.GetResourceTypeName(originalResources[0]), "yaml", resouresPkg.GetResourcesYaml(originalResources))
		}

		for _, originalResource := range originalResources {
			if err := store.SaveOriginalResource(ctx, originalResource); err != nil {
				return err
			}
		}
	}
	return nil
}

// LoadOrCreateAndSaveLastBlueprintWithValidation attempts to load last blueprint from store.
// If available, it validates whether the current host state matches that.
// If not available, it loads current host state for all resources from given target Blueprint
// and saves it.
func LoadOrCreateAndSaveLastBlueprintWithValidation(
	ctx context.Context,
	host hostPkg.Host,
	store storePkg.Store,
	targetBlueprint *blueprintPkg.Blueprint,
) (*blueprintPkg.Blueprint, error) {
	ctx, logger := log.MustContextLoggerSection(ctx, "↩️ Loading last Blueprint")
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
			_, logger := log.MustContextLoggerSection(ctx, "🧩 Loaded Blueprint")
			for _, step := range lastBlueprint.Steps {
				resources := step.Resources()
				if len(resources) == 1 {
					logger.Info(resouresPkg.GetResourceTypeName(resources[0]), "yaml", resouresPkg.GetResourceYaml(resources[0]))
				} else {
					logger.Info(resouresPkg.GetResourceTypeName(resources[0]), "yaml", resouresPkg.GetResourcesYaml(resources))
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
		chunks := diff.DiffAsYaml(lastBlueprint, currentBlueprint)

		if chunks.HaveChanges() {
			return nil, fmt.Errorf(
				"host state has changed:\n%s",
				chunks.String(),
			)
		}
	}
	return lastBlueprint, nil
}

// PrepAndPlan does all actions required to prepare a plan, computes the plan and return it.
func PrepAndPlan(
	ctx context.Context,
	host hostPkg.Host,
	store storePkg.Store,
	targetResources resouresPkg.Resources,
) (Plan, *blueprintPkg.Blueprint, *blueprintPkg.Blueprint, error) {
	targetBlueprint, err := CreateTargetBlueprint(ctx, host, targetResources)
	if err != nil {
		return nil, nil, nil, err
	}

	err = SaveOriginalResourcesState(ctx, host, store, targetBlueprint)
	if err != nil {
		return nil, nil, nil, err
	}

	lastBlueprint, err := LoadOrCreateAndSaveLastBlueprintWithValidation(
		ctx, host, store, targetBlueprint,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	plan, err := NewPlan(
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
