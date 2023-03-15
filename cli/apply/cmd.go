package apply

import (
	"fmt"

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

		// Load saved HostState
		savedHostState, err := state.LoadHostState(ctx, persistantState)
		if err != nil {
			logger.Fatal(err)
		}

		// Read current state
		var initialResources resource.Resources
		if savedHostState != nil {
			initialResources = savedHostState.Resources
		}
		initialResources = initialResources.AppendIfNotPresent(bundle.Resources())
		fmt.Printf("initialResources:\n%#v\n", initialResources)
		initialResourcesStateMap, err := resource.NewResourcesStateMap(ctx, hst, initialResources)
		if err != nil {
			logger.Fatal(err)
		}

		// Check saved HostState
		if savedHostState != nil {
			if err := savedHostState.Check(ctx, hst, initialResourcesStateMap); err != nil {
				logger.Fatal(err)
			}
		}

		// Plan
		plan, err := resource.NewPlanFromSavedStateAndBundle(
			ctx, hst, bundle, savedHostState, initialResourcesStateMap, resource.ActionNone,
		)
		if err != nil {
			logger.Fatal(err)
		}
		plan.Print(ctx)

		// Execute plan
		success := true
		err = plan.Execute(ctx, hst)

		if err == nil {
			// Save plan state
			newHostState := resource.NewHostState(bundle.Resources())
			if err := state.SaveHostState(ctx, newHostState, persistantState); err != nil {
				logger.Fatal(err)
			}
		} else {
			// Rollback
			nestedCtx := log.IndentLogger(ctx)
			nestedLogger := log.GetLogger(nestedCtx)
			success = false
			nestedLogger.Error(err)
			nestedLogger.Warn("Failed to execute plan, rolling back to previously saved state.")

			rollbackPlan, err := resource.NewRollbackPlan(nestedCtx, hst, bundle, initialResources)
			if err != nil {
				logger.Fatal(err)
			}
			rollbackPlan.Print(nestedCtx)

			err = rollbackPlan.Execute(nestedCtx, hst)
			if err != nil {
				nestedLogger.Error(err)
				logger.Fatal("Rollback failed! You may try the restore command and / or fix things manually.")
			}
			nestedLogger.Info("👌 Rollback successful.")
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
