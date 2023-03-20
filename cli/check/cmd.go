package check

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"

	"github.com/fornellas/resonance/cli/lib"
	"github.com/fornellas/resonance/state"
)

var Cmd = &cobra.Command{
	Use:   "check [flags]",
	Short: "Check host state.",
	Long:  "Loads previously saved host state and check whether it is clean or not.",
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
			logger.Fatal("No previously saved host state available to check")
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

		// Result
		logger.Info("🎆 State is clean")
	},
}

func Reset() {

}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)
}
