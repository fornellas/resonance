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

		// Host
		hst, err := lib.GetHost()
		if err != nil {
			logger.Fatal(err)
		}

		// PersistantState
		persistantState, err := lib.GetPersistantState()
		if err != nil {
			logger.Fatal(err)
		}

		// Load resources
		resourceBundles, err := resource.LoadBundles(ctx, args[0])
		if err != nil {
			logger.Fatal(err)
		}

		// Load saved state
		savedHostState, err := state.LoadHostState(ctx, persistantState)
		if err != nil {
			logger.Fatal(err)
		}

		// Plan
		plan, err := resource.NewPlanFromBundles(ctx, hst, savedHostState, resourceBundles)
		if err != nil {
			logger.Fatal(err)
		}
		plan.Print(ctx)

		// Execute plan
		newHostState, err := plan.Execute(ctx, hst)
		if err != nil {
			logger.Fatal(err)
		}

		// Save host state
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
