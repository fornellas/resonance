package main

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resources"
)

var ApplyCmd = &cobra.Command{
	Use:   "apply [flags] [file|dir]",
	Short: "Apply resources.",
	Long:  "Load resources from file/dir and apply them.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		ctx := cmd.Context()
		ctx, logger := log.MustContextLoggerWithSection(ctx, "✏️ Applying")

		host, err := GetHost(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		defer host.Close(ctx)
		logger.Info("🖥️ Target", "host", host)

		// store := GetStore(host)

		var targetStates []resources.State
		{
			ctx, _ := log.MustContextLoggerWithSection(ctx, "📂 Loading resources", "path", path)
			stateDefinitions, err := resources.LoadStatesDefinitionFromPath(ctx, path)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			for _, stateDefinition := range stateDefinitions {
				states, err := stateDefinition.States()
				if err != nil {
					logger.Error(err.Error())
					Exit(1)
				}
				targetStates = append(targetStates, states...)
			}

			// ctx, _ := log.MustContextLoggerWithSection(ctx, "📂 Loading target resources")
			// 	var err error
			// 	targetStates, err = resourcesPkg.LoadPath(ctx, path)
			// 	if err != nil {
			// 		logger.Error(err.Error())
			// 		Exit(1)
			// 	}
			// 	_, logger := log.MustContextLoggerWithSection(ctx, "📚 All loaded resources")
			// 	for _, resource := range targetStates {
			// 		logger.Info(resourcesPkg.GetResourceTypeName(resource), "yaml", resourcesPkg.GetResourceYaml(resource))
			// 	}
		}

		// var plan planPkg.Plan
		// var targetBlueprint *blueprintPkg.Blueprint
		// var lastBlueprint *blueprintPkg.Blueprint
		// {
		// 	ctx, logger := log.MustContextLoggerWithSection(ctx, "📝 Planning")
		// 	plan, targetBlueprint, lastBlueprint, err = planPkg.PrepAndPlan(ctx, host, store, targetStates)
		// 	if err != nil {
		// 		logger.Error(err.Error())
		// 		Exit(1)
		// 	}
		// 	{
		// 		ctx, _ := log.MustContextLoggerWithSection(ctx, "💾 Saving tatrget Blueprint")
		// 		hasTargetBlueprint, err := store.HasTargetBlueprint(ctx)
		// 		if err != nil {
		// 			logger.Error(err.Error())
		// 			Exit(1)
		// 		}
		// 		if hasTargetBlueprint {
		// 			logger.Error("a previous apply was interrupted")
		// 			Exit(1)
		// 		} else {
		// 			if err := store.SaveTargetBlueprint(ctx, targetBlueprint); err != nil {
		// 				logger.Error(err.Error())
		// 				Exit(1)
		// 			}
		// 		}
		// 	}
		// }

		// if err := plan.Apply(ctx, host); err != nil {
		// 	logger.Error(err.Error())
		// 	Exit(1)
		// }

		// {
		// 	ctx, logger := log.MustContextLoggerWithSection(ctx, "🧹 State cleanup")

		// 	targetStatesMap := resourcesPkg.NewResourceMap(targetStates)
		// 	for _, lastResource := range lastBlueprint.Resources() {
		// 		if !targetStatesMap.HasResourceWithSameTypeId(lastResource) {
		// 			if err := store.DeleteOriginalResource(ctx, lastResource); err != nil {
		// 				logger.Error(err.Error())
		// 				Exit(1)
		// 			}
		// 		}
		// 	}

		// 	if err := store.SaveLastBlueprint(ctx, targetBlueprint); err != nil {
		// 		logger.Error(err.Error())
		// 		Exit(1)
		// 	}

		// 	if err := store.DeleteTargetBlueprint(ctx); err != nil {
		// 		logger.Error(err.Error())
		// 		Exit(1)
		// 	}
		// }

		// logger.Info("🎆 Apply successful")
	},
}

func init() {
	AddTargetFlags(ApplyCmd)

	AddStoreFlags(ApplyCmd)

	RootCmd.AddCommand(ApplyCmd)
}
