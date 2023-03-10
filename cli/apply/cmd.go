package apply

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/log"

	"github.com/fornellas/resonance/cli/lib"
	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/state"
)

var Cmd = &cobra.Command{
	Use:   "apply [flags] resources_root",
	Short: "Applies configuration to a host.",
	Long:  "Loads all resoures from .yaml files at resources_root, the previous state, craft a plan and applies required changes to given host.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		logger := log.GetLogger(ctx)
		nestedCtx := log.IndentLogger(ctx)
		nestedLogger := log.GetLogger(nestedCtx)

		// Host
		hst, err := lib.GetHost()
		if err != nil {
			logger.Fatal(err)
		}

		// PersistantState
		persistantState, err := lib.GetPersistantState(hst)
		if err != nil {
			logger.Fatal(err)
		}

		// Load resources
		root := args[0]
		bundles, err := resource.LoadBundles(ctx, root)
		if err != nil {
			logger.Fatal(err)
		}

		// Load saved state
		savedHostState, err := state.LoadHostState(ctx, persistantState)
		if err != nil {
			logger.Fatal(err)
		}
		if savedHostState != nil {
			if err := savedHostState.Validate(ctx, hst); err != nil {
				logger.Fatal(err)
			}
		}

		// Read bundle state
		var excludeResourceKeys []resource.ResourceKey
		if savedHostState != nil {
			excludeResourceKeys = savedHostState.ResourceKeys()
		}
		bundlesHostState, err := bundles.GetHostState(ctx, hst, excludeResourceKeys)
		if err != nil {
			logger.Fatal(err)
		}

		// Initial state
		initialHostState := bundlesHostState
		if savedHostState != nil {
			initialHostState = bundlesHostState.Merge(*savedHostState)
		}

		// Plan
		plan, err := resource.NewPlanFromSavedStateAndBundles(ctx, hst, bundles, savedHostState)
		if err != nil {
			logger.Fatal(err)
		}
		plan.Print(ctx)

		// Execute plan
		success := true
		planHostState, err := plan.Execute(ctx, hst)
		if err != nil {
			// Rollback
			success = false
			nestedLogger.Error(err)
			nestedLogger.Warn("Failed to execute plan, rolling back to previously saved state.")

			rollbackPlan, err := resource.NewRollbackPlan(
				nestedCtx, hst, bundles, savedHostState, initialHostState,
			)
			if err != nil {
				logger.Fatal(err)
			}
			rollbackPlan.Print(nestedCtx)

			planHostState, err = rollbackPlan.Execute(nestedCtx, hst)
			if err != nil {
				nestedLogger.Error(err)
				logger.Fatal("Rollback failed!")
			}
			nestedLogger.Info("👌 Rollback successful.")
		}

		// Save host state
		if err := state.SaveHostState(ctx, planHostState, persistantState); err != nil {
			logger.Fatal(err)
		}

		// Result
		if success {
			logger.Info("🎆 Success")
		} else {
			logger.Fatal("Failed to apply, rollback to previously saved state successful.")
		}
	},
}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)
}
