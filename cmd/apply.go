package main

import (
	"github.com/spf13/cobra"

	blueprintPkg "github.com/fornellas/resonance/blueprint"
	"github.com/fornellas/resonance/log"
	planPkg "github.com/fornellas/resonance/plan"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

var ApplyCmd = &cobra.Command{
	Use:   "apply [flags] [file|dir]",
	Short: "Apply resources.",
	Long:  "Load resources from file/dir and apply them.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		ctx := cmd.Context()

		logger := log.MustLogger(ctx)

		logger.Info("✏️ Applying", "path", path)

		host, err := GetHost(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		defer host.Close(ctx)
		logger.Info("🖥️ Target", "host", host)

		store := GetStore(host)

		var targetResources resourcesPkg.Resources
		{
			var err error
			ctx, _ := log.MustContextLoggerWithSection(ctx, "📂 Loading target resources")
			targetResources, err = resourcesPkg.LoadPath(ctx, path)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			_, logger := log.MustContextLoggerWithSection(ctx, "📚 All loaded resources")
			for _, resource := range targetResources {
				logger.Info(resourcesPkg.GetResourceTypeName(resource), "yaml", resourcesPkg.GetResourceYaml(resource))
			}
		}

		var plan planPkg.Plan
		var targetBlueprint *blueprintPkg.Blueprint
		var lastBlueprint *blueprintPkg.Blueprint
		{
			ctx, logger := log.MustContextLoggerWithSection(ctx, "📝 Planning")
			plan, targetBlueprint, lastBlueprint, err = planPkg.PrepAndPlan(ctx, host, store, targetResources)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			{
				ctx, _ := log.MustContextLoggerWithSection(ctx, "💾 Saving tatrget Blueprint")
				hasTargetBlueprint, err := store.HasTargetBlueprint(ctx)
				if err != nil {
					logger.Error(err.Error())
					Exit(1)
				}
				if hasTargetBlueprint {
					logger.Error("a previous apply was interrupted")
					Exit(1)
				} else {
					if err := store.SaveTargetBlueprint(ctx, targetBlueprint); err != nil {
						logger.Error(err.Error())
						Exit(1)
					}
				}
			}
		}

		if err := plan.Apply(ctx, host); err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		{
			ctx, logger := log.MustContextLoggerWithSection(ctx, "🧹 State cleanup")

			targetResourcesMap := resourcesPkg.NewResourceMap(targetResources)
			for _, lastResource := range lastBlueprint.Resources() {
				if !targetResourcesMap.HasResourceWithSameTypeId(lastResource) {
					if err := store.DeleteOriginalResource(ctx, lastResource); err != nil {
						logger.Error(err.Error())
						Exit(1)
					}
				}
			}

			if err := store.SaveLastBlueprint(ctx, targetBlueprint); err != nil {
				logger.Error(err.Error())
				Exit(1)
			}

			if err := store.DeleteTargetBlueprint(ctx); err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
		}

		logger.Info("🎆 Apply successful")
	},
}

func init() {
	AddTargetFlags(ApplyCmd)

	AddStoreFlags(ApplyCmd)

	RootCmd.AddCommand(ApplyCmd)
}
