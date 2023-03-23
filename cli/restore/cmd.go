package restore

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/cli/lib"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/state"
)

var Cmd = &cobra.Command{
	Use:   "restore [flags]",
	Short: "Restore host state to previously saved state.",
	Long:  "Loads previously saved state for host and applies it again.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		logger := log.GetLogger(ctx)

		// Host
		hst, err := lib.GetHost(ctx)
		if err != nil {
			logger.Fatal(err)
		}
		defer hst.Close()

		// PersistantState
		persistantState, err := lib.GetPersistantState(hst)
		if err != nil {
			logger.Fatal(err)
		}

		// Load saved HostState
		hostState, err := state.LoadHostState(ctx, persistantState)
		if err != nil {
			logger.Fatal(err)
		}
		if hostState == nil {
			logger.Fatal("No previously saved host state available to restore from")
		}

		// Read state
		typeNameStateMap, err := resource.GetTypeNameStateMap(
			ctx, hst, hostState.PreviousBundle.TypeNames(), true,
		)
		if err != nil {
			logger.Fatal(err)
		}

		// Rollback Bundle
		rollbackBundle := resource.NewRollbackBundle(
			hostState.PreviousBundle, nil, typeNameStateMap, resource.ActionConfigure,
		)

		// Plan
		plan, err := resource.NewPlan(
			ctx, hst, hostState.PreviousBundle, nil, typeNameStateMap, resource.ActionConfigure,
		)
		if err != nil {
			logger.Fatal(err)
		}
		plan.Print(ctx, hst)

		// Save rollback bundle
		if err := state.SaveHostState(
			ctx, resource.NewHostState(rollbackBundle, true), persistantState,
		); err != nil {
			logger.Fatal(err)
		}

		// Execute
		if err = plan.Execute(ctx, hst); err == nil {
			if err := state.SaveHostState(
				ctx, resource.NewHostState(hostState.PreviousBundle, false), persistantState,
			); err != nil {
				logger.Fatal(err)
			}

			logger.Info("🎆 Restore successful")
		} else {
			nestedCtx := log.IndentLogger(ctx)
			nestedLogger := log.GetLogger(nestedCtx)
			nestedLogger.Error(err)

			// Rollback
			if err := lib.Rollback(ctx, hst, rollbackBundle); err != nil {
				logger.Fatal("Rollback failed! You may try the 'rollback' command or fix things manually.")
			}

			// Save State
			if err := state.SaveHostState(
				ctx, resource.NewHostState(hostState.PreviousBundle, false), persistantState,
			); err != nil {
				logger.Fatal(err)
			}

			logger.Fatal("Failed, rollback to previously saved state successful.")
		}
	},
}

func Reset() {

}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)
}
