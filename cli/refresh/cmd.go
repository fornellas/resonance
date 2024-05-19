package refresh

// import (
// 	"github.com/spf13/cobra"

// 	"github.com/fornellas/resonance/log"
// 	"github.com/fornellas/resonance/resource"
// 	"github.com/fornellas/resonance/state"

// 	"github.com/fornellas/resonance/cli/lib"
// )

// var Cmd = &cobra.Command{
// 	Use:   "refresh [flags]",
// 	Short: "Update saved host state with current host state.",
// 	Long:  "Loads previously saved host state and update each resource state with the current host state. In general, it is a bad idea to use this command, as it will be committing external changes to the host state.",
// 	Args:  cobra.ExactArgs(0),
// 	Run: func(cmd *cobra.Command, args []string) {
// 		ctx := cmd.Context()

// 		logger := log.GetLogger(ctx)

// 		// Host
// 		hst, err := lib.GetHost(ctx)
// 		if err != nil {
// 			logger.Fatal(err)
// 		}
// 		defer hst.Close()

// 		// PersistantState
// 		persistantState, err := lib.GetPersistantState(hst)
// 		if err != nil {
// 			logger.Fatal(err)
// 		}

// 		// Load saved HostState
// 		hostState, err := state.LoadHostState(ctx, persistantState)
// 		if err != nil {
// 			logger.Fatal(err)
// 		}
// 		if hostState == nil {
// 			logger.Fatal("No previously saved host state available to check")
// 		}

// 		// Read state
// 		typeNameStateMap, err := resource.GetTypeNameStateMap(
// 			ctx, hst, hostState.PreviousBundle.TypeNames(), true,
// 		)
// 		if err != nil {
// 			logger.Fatal(err)
// 		}

// 		// Refresh
// 		newHostState, err := hostState.Refresh(ctx, typeNameStateMap)
// 		if err != nil {
// 			logger.Fatal(err)
// 		}

// 		// Save state
// 		if err := state.SaveHostState(ctx, newHostState, persistantState); err != nil {
// 			logger.Fatal(err)
// 		}

// 		// Success
// 		logger.Info("🎆 Refresh successful")
// 	},
// }

// func Reset() {

// }

// func init() {
// 	lib.AddHostFlags(Cmd)
// 	lib.AddPersistantStateFlags(Cmd)
// }
