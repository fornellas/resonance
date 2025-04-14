package main

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/log"
	planPkg "github.com/fornellas/resonance/plan"
	resourcesPkg "github.com/fornellas/resonance/resources"
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

		logger.Info("✏️ Planning", "path", path)

		host, err := GetHost(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		defer host.Close(ctx)
		logger.Info("🖥️ Target", "host", host)

		store, _ := GetStore(host)

		// Load Target Resources
		var targetResources resourcesPkg.Resources
		{
			var err error
			ctx, _ := log.WithGroup(ctx, "📂 Loading target resources")
			targetResources, err = resourcesPkg.LoadPath(ctx, path)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			_, logger := log.WithGroup(ctx, "📚 All loaded resources")
			for _, resource := range targetResources {
				logger.Info(resourcesPkg.GetResourceTypeName(resource), "yaml", resourcesPkg.GetResourceYaml(resource))
			}
		}

		var plan planPkg.Plan
		{
			ctx, logger := log.WithGroup(ctx, "📝 Planning")
			plan, _, _, err = planPkg.PrepAndPlan(ctx, host, store, targetResources)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
		}

		{
			_, logger := log.WithGroup(ctx, "💡 Actions")
			for _, action := range plan {
				args := []any{}
				diffStr := action.DiffString()
				if len(diffStr) > 0 {
					args = append(args, []any{"diff", diffStr}...)
				}
				logger.Info(action.String(), args...)
			}
		}
		logger.Info("🎆 Planning successful")
	},
}

func init() {
	AddTargetFlags(PlanCmd)

	AddStoreFlags(PlanCmd)

	RootCmd.AddCommand(PlanCmd)
}
