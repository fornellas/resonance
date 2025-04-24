package main

import (
	"fmt"
	"log/slog"

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

		var logger *slog.Logger
		ctx, logger = log.WithGroupAttrs(ctx, "✏️ Apply", "path", path)

		host, err := GetHost(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		defer func() {
			if err := host.Close(ctx); err != nil {
				logger.Error("failed to close host", "error", err)
			}
		}()
		ctx, _ = log.WithAttrs(ctx, "host", fmt.Sprintf("%s => %s", host.Type(), host.String()))

		store, storeConfig := GetStore(host)
		ctx, logger = log.WithAttrs(ctx, "store", fmt.Sprintf("%s %s", storeValue.String(), storeConfig))

		var targetResources resourcesPkg.Resources
		{
			var err error
			targetResources, err = resourcesPkg.LoadPath(ctx, path)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			_, logger := log.WithGroup(ctx, "📚 Target resources")
			for _, resource := range targetResources {
				logger.Debug(resourcesPkg.GetResourceTypeName(resource), "yaml", resourcesPkg.GetResourceYaml(resource))
			}
		}

		var plan planPkg.Plan
		var targetBlueprint *blueprintPkg.Blueprint
		var lastBlueprint *blueprintPkg.Blueprint
		{
			plan, targetBlueprint, lastBlueprint, err = planPkg.CraftPlan(ctx, host, store, targetResources)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			{
				ctx, _ := log.WithGroup(ctx, "💾 Saving target Blueprint")
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
			ctx, logger := log.WithGroup(ctx, "🧹 State cleanup")

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
	AddHostFlags(ApplyCmd)

	AddStoreFlags(ApplyCmd)

	RootCmd.AddCommand(ApplyCmd)
}
