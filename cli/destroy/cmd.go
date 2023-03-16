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

		// Load saved state
		savedHostState, err := state.LoadHostState(ctx, persistantState)
		if err != nil {
			logger.Fatal(err)
		}
		if savedHostState == nil {
			logger.Fatal("No previously saved state to destroy")
		}

		// Read current state
		initialResourcesStateMap, err := resource.GetResourcesStateMap(
			ctx, hst, savedHostState.Bundle.Resources(),
		)
		if err != nil {
			logger.Fatal(err)
		}

		// Plan
		plan, err := resource.NewPlanFromSavedStateAndBundle(
			ctx, hst, savedHostState.Bundle, nil, initialResourcesStateMap, resource.ActionDestroy,
		)
		if err != nil {
			logger.Fatal(err)
		}
		plan.Print(ctx)

		// Execute plan
		err = plan.Execute(ctx, hst)
		if err != nil {
			logger.Fatal(err)
		}

		// Save state
		newHostState := resource.NewHostState(resource.Bundle{})
		if err := state.SaveHostState(ctx, newHostState, persistantState); err != nil {
			logger.Fatal(err)
		}

		// Success
		logger.Info("🎆 Success")
	},
}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)
}
