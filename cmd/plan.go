package main

import (
	"errors"
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

		var retErr error
		defer func() {
			if retErr != nil {
				logger.Error("Failed", "err", retErr)
				Exit(1)
			}
		}()

		host, err := GetHost(ctx)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to get host: %w", err))
			return
		}
		defer func() {
			if err := host.Close(ctx); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to close host: %w", err))
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
				retErr = errors.Join(retErr, fmt.Errorf("failed to load resources: %w", err))
				return
			}
			_, logger := log.MustWithGroup(ctx, "📚 Target resources")
			for _, resource := range targetResources {
				logger.Debug(resourcesPkg.GetResourceTypeName(resource), "yaml", resourcesPkg.GetResourceYaml(resource))
			}
		}

		var plan planPkg.Plan
		plan, _, _, err = planPkg.CraftPlan(ctx, host, store, targetResources)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to plan: %w", err))
			return
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
