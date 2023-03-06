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
	Long:  "Loads previous state and check whether it is clean or not.",
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
		persistantState, err := lib.GetPersistantState()
		if err != nil {
			logger.Fatal(err)
		}

		// Load saved state
		savedHostState, err := state.LoadHostState(ctx, persistantState)
		if err != nil {
			logger.Fatal(err)
		}

		// Check
		checkResults, err := savedHostState.Check(ctx, hst)
		if err != nil {
			logger.Fatal(err)
		}
		ok := true
		for _, checkResult := range checkResults {
			if checkResult != resource.CheckResult(true) {
				ok = false
			}
		}
		if ok {
			logger.Info("🎆 State is OK")
		} else {
			logger.Fatal("State has changed!")
		}
	},
}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)
}
