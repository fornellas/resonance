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
		resourceBundles, err := resource.LoadBundles(ctx, args[0])
		if err != nil {
			logger.Fatal(err)
		}

		// Load saved state
		logger.Info("📋 Assessing host state")
		hasPreviousHostState, initialHostState, err := state.LoadUpdateHostState(
			nestedCtx, persistantState, resourceBundles, hst,
		)
		if err != nil {
			logger.Fatal(err)
		}
		if hasPreviousHostState {
			if err := initialHostState.Validate(nestedCtx, hst); err != nil {
				logger.Fatal(err)
			}
		}

		// Plan
		var previousHostState *resource.HostState
		if hasPreviousHostState {
			previousHostState = &initialHostState
		}
		plan, err := resource.NewPlanFromBundles(ctx, hst, previousHostState, resourceBundles)
		if err != nil {
			logger.Fatal(err)
		}
		plan.Print(ctx)

		// Execute plan
		success := true
		newHostState, err := plan.Execute(ctx, hst)
		if err != nil {
			success = false
			nestedLogger.Error(err)
			nestedLogger.Warn("Failed to execute plan, rolling back to previously saved state.")

			plan, err := resource.NewActionPlanFromHostState(
				nestedCtx, hst, &initialHostState, resource.ActionApply,
			)
			if err != nil {
				logger.Fatal(err)
			}
			plan.Print(nestedCtx)

			// Execute plan
			newHostState, err = plan.Execute(nestedCtx, hst)
			if err != nil {
				nestedLogger.Error(err)
				logger.Fatal("Rollback failed!")
			}
			nestedLogger.Info("👌 Rollback successful.")
		}

		// Save host state
		if err := state.SaveHostState(ctx, newHostState, persistantState); err != nil {
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
