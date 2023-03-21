package rollback

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/cli/lib"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/state"
)

var Cmd = &cobra.Command{
	Use:   "rollback [flags]",
	Short: "Rollback host state after a partial action.",
	Long:  "If an action (eg: apply, destroy etc) fails mid-way, the saved host state will still contain the rollback plan. This command enables to complete the rollback.",
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
			logger.Fatal("No previously saved host state available to rollback from")
		}
		if hostState.RollbackBundle == nil {
			logger.Fatal("No rollback required for saved host state")
		}

		// Read state
		typeNameStateMap, err := resource.GetTypeNameStateMap(
			ctx, hst, hostState.PreviousBundle.TypeNames(),
		)
		if err != nil {
			logger.Fatal(err)
		}

		// Plan
		plan, err := resource.NewPlan(
			ctx, hst, *hostState.RollbackBundle, nil, typeNameStateMap, resource.ActionConfigure,
		)
		if err != nil {
			logger.Fatal(err)
		}
		plan.Print(ctx, hst)

		// Execute
		if err = plan.Execute(ctx, hst); err == nil {
			if err := state.SaveHostState(
				ctx, resource.NewHostState(hostState.PreviousBundle, nil), persistantState,
			); err != nil {
				logger.Fatal(err)
			}

			logger.Info("🎆 Rollback successful")
		} else {
			logger.Fatal("Failed to rollback.")
		}
	},
}

func Reset() {

}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)
}
