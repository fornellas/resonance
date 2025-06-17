package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fornellas/slogxt/log"

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

		ctx, logger := log.MustWithGroupAttrs(cmd.Context(), "📝 Planning", "path", path)

		host, err := GetHost(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		defer func() {
			if err := host.Close(ctx); err != nil {
				logger.Error("failed to close host", "error", err)
				Exit(1)
			}
		}()
		ctx, _ = log.MustWithAttrs(ctx, "host", fmt.Sprintf("%s => %s", host.Type(), host.String()))

		store, storeConfig := GetStore(host)
		ctx, logger = log.MustWithAttrs(ctx, "store", fmt.Sprintf("%s %s", storeValue.String(), storeConfig))

		var targetResources resourcesPkg.Resources
		{
			var err error
			targetResources, err = resourcesPkg.LoadPath(ctx, path)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			_, logger := log.MustWithGroup(ctx, "📚 Target resources")
			for _, resource := range targetResources {
				logger.Debug(resourcesPkg.GetResourceTypeName(resource), "yaml", resourcesPkg.GetResourceYaml(resource))
			}
		}

		var plan planPkg.Plan
		plan, _, _, err = planPkg.CraftPlan(ctx, host, store, targetResources)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		{
			_, logger := log.MustWithGroup(ctx, "💡 Actions")
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
