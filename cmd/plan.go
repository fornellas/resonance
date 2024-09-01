package main

import (
	"github.com/spf13/cobra"

	planPkg "github.com/fornellas/resonance/internal/plan"
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

		logger.Info("✏️ Planning", "path", path)

		host, err := GetHost(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		defer host.Close()
		logger.Info("🖥️ Target", "host", host)

		store := GetStore(host)

		// Load Target Resources
		var targetResources resouresPkg.Resources
		{
			var err error
			ctx, _ := log.MustContextLoggerSection(ctx, "📂 Loading target resources")
			targetResources, err = iResouresPkg.LoadPath(ctx, path)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			_, logger := log.MustContextLoggerSection(ctx, "📚 All loaded resources")
			for _, resource := range targetResources {
				logger.Info(resouresPkg.GetResourceTypeName(resource), "yaml", resouresPkg.GetResourceYaml(resource))
			}
		}

		var plan planPkg.Plan
		{
			ctx, logger := log.MustContextLoggerSection(ctx, "📝 Planning")
			plan, _, _, err = planPkg.PrepAndPlan(ctx, host, store, targetResources)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
		}

		{
			_, logger := log.MustContextLoggerSection(ctx, "💡 Actions")
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
	AddHostFlags(PlanCmd)

	AddStoreFlags(PlanCmd)

	RootCmd.AddCommand(PlanCmd)
}
