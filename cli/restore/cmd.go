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
	Short: "Restore host to previously saved state.",
	Long:  "Loads previous state from host and applies all of them.",
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
		savedHostState, err := state.LoadHostState(ctx, persistantState)
		if err != nil {
			logger.Fatal(err)
		}
		if savedHostState == nil {
			logger.Fatal("No previously saved state to restore from")
		}

		// Plan
		rollbackPlan, err := resource.NewRollbackPlan(
			ctx, hst, resource.Bundle{}, savedHostState.Bundle.Resources(),
		)
		if err != nil {
			logger.Fatal(err)
		}
		rollbackPlan.Print(ctx)

		// Execute
		err = rollbackPlan.Execute(ctx, hst)
		if err != nil {
			logger.Fatal(err)
		}

		// Result
		logger.Info("🎆 Success")
	},
}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)
}
