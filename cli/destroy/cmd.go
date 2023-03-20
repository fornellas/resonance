package destroy

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/cli/lib"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/state"
)

var Cmd = &cobra.Command{
	Use:   "destroy [flags]",
	Short: "Destroy previously configured resources.",
	Long:  "Loads previous state from host and destroys all of them from the host.",
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
			logger.Fatal("No previously saved host state available to be destroyed")
		}

		// Read state
		typeNameStateMap, err := resource.GetTypeNameStateMap(
			ctx, hst, hostState.PreviousBundle.TypeNames(),
		)
		if err != nil {
			logger.Fatal(err)
		}

		// Check saved HostState
		if hostState != nil {
			isClean, err := hostState.PreviousBundle.IsClean(ctx, hst, typeNameStateMap)
			if err != nil {
				logger.Fatal(err)
			}
			if !isClean {
				logger.Fatalf(
					"Host state is not clean: this often means external agents altered the host state after previous apply. Try the 'refresh' or 'restore' commands.",
				)
			}
		}

		// Rollback NewRollbackBundle
		rollbackBundle := resource.NewRollbackBundle(
			hostState.PreviousBundle, nil, typeNameStateMap, resource.ActionConfigure,
		)

		// TODO save rollback bundle

		// Plan
		plan, err := resource.NewPlan(
			ctx, hst, hostState.PreviousBundle, nil, typeNameStateMap, resource.ActionDestroy,
		)
		if err != nil {
			logger.Fatal(err)
		}
		plan.Print(ctx, hst)

		// Execute plan
		err = plan.Execute(ctx, hst)

		if err == nil {
			// Save plan state
			newHostState := resource.NewHostState(resource.Bundle{})
			if err := state.SaveHostState(ctx, newHostState, persistantState); err != nil {
				logger.Fatal(err)
			}

			logger.Info("🎆 Success")
		} else {
			nestedCtx := log.IndentLogger(ctx)
			nestedLogger := log.GetLogger(nestedCtx)
			nestedLogger.Error(err)
			lib.Rollback(ctx, hst, rollbackBundle)
		}
	},
}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)
}
