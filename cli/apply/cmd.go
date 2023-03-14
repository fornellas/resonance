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
		bundle, err := resource.LoadBundle(ctx, root)
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
		} else {
			nestedLogger.Warn("No previously saved state available.")
		}

		// Read bundle state
		var excludeTypeNames []resource.TypeName
		if savedHostState != nil {
			excludeTypeNames = savedHostState.TypeNames()
		}
		bundlesHostState, err := bundle.GetHostState(ctx, hst, excludeTypeNames)
		if err != nil {
			logger.Fatal(err)
		}

		// Initial state
		initialHostState := bundlesHostState
		if savedHostState != nil {
			initialHostState = bundlesHostState.Append(*savedHostState)
		}

		// Save initial state
		// TODO requires marking (bundle - saved) state as "do not destroy"
		// if err := state.SaveHostState(ctx, initialHostState, persistantState); err != nil {
		// 	logger.Fatal(err)
		// }

		// Plan
		plan, err := resource.NewPlanFromSavedStateAndBundle(
			ctx, hst, bundle, savedHostState, resource.ActionNone,
		)
		if err != nil {
			logger.Fatal(err)
		}
		plan.Print(ctx)

		// Execute plan
		success := true
		planHostState, err := plan.Execute(ctx, hst)

		// Rollback
		if err != nil {
			success = false
			nestedLogger.Error(err)
			nestedLogger.Warn("Failed to execute plan, rolling back to previously saved state.")

			rollbackPlan, err := resource.NewRollbackPlan(
				nestedCtx, hst, bundle, initialHostState,
			)
			if err != nil {
				logger.Fatal(err)
			}
			rollbackPlan.Print(nestedCtx)

			planHostState, err = rollbackPlan.Execute(nestedCtx, hst)
			if err != nil {
				nestedLogger.Error(err)
				logger.Fatal("Rollback failed! You may try the restore command and / or fix things manually.")
			}
			nestedLogger.Info("👌 Rollback successful.")
		}

		// Save plan state
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
