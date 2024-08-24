package main

import (
	"fmt"

	"github.com/spf13/cobra"

	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
	"github.com/fornellas/resonance/internal/diff"
	iResouresPkg "github.com/fornellas/resonance/internal/resources"
	"github.com/fornellas/resonance/log"
	resouresPkg "github.com/fornellas/resonance/resources"
)

var PlanCmd = &cobra.Command{
	Use:   "plan [flags] [file|dir]",
	Short: "Plan actions.",
	Long:  "Load resources from file/dir and plan which actions are required to apply them.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		ctx := cmd.Context()

		logger := log.MustLogger(ctx)

		logger.Info("✏️ Planning", "[file|dir]", path)

		hst, err := GetHost(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		defer hst.Close()
		logger.Info("🖥️ Target", "host", hst)

		// Load Target Resources
		var targetResources resouresPkg.Resources
		{
			ctx, logger := log.MustContextLoggerSection(ctx, "📂 Loading target resources")
			targetResources, err = iResouresPkg.LoadPath(ctx, path)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			_, logger = log.MustContextLoggerSection(ctx, "📚 Resources")
			for _, resource := range targetResources {
				logger.Info(resouresPkg.GetResourceTypeName(resource), "yaml", resouresPkg.GetResourceYaml(resource))
			}
		}

		// Compute Target Blueprint
		var targetBlueprint *blueprintPkg.Blueprint
		{
			ctx, logger := log.MustContextLoggerSection(ctx, "⚙️ Computing target Blueprint")
			targetBlueprint, err = blueprintPkg.NewBlueprintFromResources(ctx, targetResources)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			if err := targetBlueprint.Resolve(ctx, hst); err != nil {
				logger.Error(err.Error())
				Exit(1)
			}

			_, logger = log.MustContextLoggerSection(ctx, "🧩 Target Blueprint")
			for _, step := range targetBlueprint.Steps {
				resources := step.Resources()
				if len(resources) == 1 {
					logger.Info(resouresPkg.GetResourceTypeName(resources[0]), "yaml", resouresPkg.GetResourceYaml(resources[0]))
				} else {
					logger.Info(resouresPkg.GetResourceTypeName(resources[0]), "yaml", resouresPkg.GetResourcesYaml(resources))
				}
			}
		}

		store := GetStore(hst)

		// Save Target Blueprint
		{
			ctx, logger := log.MustContextLoggerSection(ctx, "🎯 Saving tatrget Blueprint")
			hasTargetBlueprint, err := store.HasTargetBlueprint(ctx)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			if hasTargetBlueprint {
				// TODO commands for retry and rollback
				logger.Error(
					"A previous apply was interrupted",
				)
				Exit(1)
			} else {
				// FIXME this should be only on apply
				if err := store.SaveTargetBlueprint(ctx, targetBlueprint); err != nil {
					logger.Error(err.Error())
					Exit(1)
				}
			}
		}

		// Original Resource
		{
			ctx, logger := log.MustContextLoggerSection(ctx, "🌱 Saving original resource states")
			for _, step := range targetBlueprint.Steps {
				noOriginalResources := resouresPkg.Resources{}
				for _, resource := range step.Resources() {
					resource = resouresPkg.NewResourceWithSameId(resource)
					hasOriginal, err := store.HasOriginalResource(ctx, resource)
					if err != nil {
						logger.Error(err.Error())
						Exit(1)
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
				originalStep, err := noOriginalStep.Load(ctx, hst)
				if err != nil {
					logger.Error(err.Error())
					Exit(1)
				}

				originalResources := originalStep.Resources()
				if len(originalResources) == 1 {
					logger.Info(resouresPkg.GetResourceTypeName(originalResources[0]), "yaml", resouresPkg.GetResourceYaml(originalResources[0]))
				} else {
					logger.Info(resouresPkg.GetResourceTypeName(originalResources[0]), "yaml", resouresPkg.GetResourcesYaml(originalResources))
				}

				for _, originalResource := range originalResources {
					if err := store.SaveOriginalResource(ctx, originalResource); err != nil {
						logger.Error(err.Error())
						Exit(1)
					}
				}
			}
		}

		// Last Blueprint
		var lastBlueprint *blueprintPkg.Blueprint
		{
			ctx, logger := log.MustContextLoggerSection(ctx, "↩️ Loading last Blueprint")
			lastBlueprint, err = store.LoadLastBlueprint(ctx)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			if lastBlueprint == nil {
				logger.Info("🔎 No last Blueprint, loading current state")
				var err error
				lastBlueprint, err = targetBlueprint.Load(ctx, hst)
				if err != nil {
					logger.Error(err.Error())
					Exit(1)
				}

				{
					_, logger := log.MustContextLoggerSection(ctx, "Loaded Blueprint")
					for _, step := range lastBlueprint.Steps {
						resources := step.Resources()
						if len(resources) == 1 {
							logger.Info(resouresPkg.GetResourceTypeName(resources[0]), "yaml", resouresPkg.GetResourceYaml(resources[0]))
						} else {
							logger.Info(resouresPkg.GetResourceTypeName(resources[0]), "yaml", resouresPkg.GetResourcesYaml(resources))
						}
					}
				}

				// FIXME this should be only on apply
				logger.Info("Saving as last Blueprint")
				if err := store.SaveLastBlueprint(ctx, lastBlueprint); err != nil {
					logger.Error(err.Error())
					Exit(1)
				}
			} else {
				logger.Info("🔎 Validating previous host state")
				currentBlueprint, err := lastBlueprint.Load(ctx, hst)
				if err != nil {
					logger.Error(err.Error())
					Exit(1)
				}
				chunks := diff.DiffAsYaml(lastBlueprint, currentBlueprint)

				if chunks.HaveChanges() {
					logger.Error(
						"host state has changed since last time, can't proceed",
						"diff", chunks.String(),
					)
					Exit(1)
				}
			}
		}

		// Plan
		{
			ctx, logger := log.MustContextLoggerSection(ctx, "📝 Planned actions")
			plan, err := blueprintPkg.NewPlan(
				ctx, hst,
				targetBlueprint,
				lastBlueprint,
				store.LoadOriginalResource,
			)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}

			actions := plan.GetActions()
			for _, action := range actions {
				if len(action.ResourceActions) == 1 {
					resourceAction := action.ResourceActions[0]
					diffArgs := []any{}
					diffStr := resourceAction.Diff().String()
					if len(diffStr) > 0 {
						diffArgs = []any{"diff", diffStr}
					}
					logger.Info(
						fmt.Sprintf("%s %s", resourceAction.Emoji, action.String()),
						diffArgs...,
					)
				} else {
					_, logger := log.MustContextLoggerSection(ctx, action.String())
					for _, resourceAction := range action.ResourceActions {
						diffArgs := []any{}
						diffStr := resourceAction.Diff().String()
						if len(diffStr) > 0 {
							diffArgs = []any{"diff", diffStr}
						}
						logger.Info(resourceAction.String(), diffArgs...)
					}
				}
			}
		}
		logger.Info("🎆 Planning successful")

		// TODO apply
		// TODO remove target state on success
	},
}

func init() {
	AddHostFlags(PlanCmd)

	AddStoreFlags(PlanCmd)

	RootCmd.AddCommand(PlanCmd)
}
