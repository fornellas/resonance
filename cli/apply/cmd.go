package apply

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/cli/lib"
	"github.com/fornellas/resonance/log"
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

		root := args[0]

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

		// Plan
		newBundle, plan, rollbackBundle := lib.Plan(ctx, hst, persistantState, root)

		// Save rollback bundle
		if err := state.SaveHostState(
			ctx, resource.NewHostState(rollbackBundle, true), persistantState,
		); err != nil {
			logger.Fatal(err)
		}

		// Execute
		if err = plan.Execute(ctx, hst); err == nil {
			if err := state.SaveHostState(
				ctx, resource.NewHostState(newBundle, false), persistantState,
			); err != nil {
				logger.Fatal(err)
			}

			logger.Info("🎆 Apply successful")
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
				ctx, resource.NewHostState(rollbackBundle, false), persistantState,
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
