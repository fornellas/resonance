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
		hst, err := lib.GetHost()
		if err != nil {
			logger.Fatal(err)
		}

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
			ctx, hst, hostState.PreviousBundle.TypeNames(),
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

		// TODO save rollback bundle

		// Execute
		if err = plan.Execute(ctx, hst); err == nil {
			newHostState := resource.NewHostState(hostState.PreviousBundle)
			if err := state.SaveHostState(ctx, newHostState, persistantState); err != nil {
				logger.Fatal(err)
			}

			logger.Info("🎆 Restore successful")
		} else {
			nestedCtx := log.IndentLogger(ctx)
			nestedLogger := log.GetLogger(nestedCtx)
			nestedLogger.Error(err)
			lib.Rollback(ctx, hst, rollbackBundle)
		}
	},
}

func Reset() {

}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)
}
