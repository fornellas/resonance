package destroy

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/cli/lib"
)

var Cmd = &cobra.Command{
	Use:   "destroy [flags]",
	Short: "Destroy previously configured resources.",
	Long:  "Loads previous state from host and destroys all of them from the host.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		panic("destroy")
		// ctx := cmd.Context()

		// logger := log.GetLogger(ctx)

		// // Host
		// hst, err := lib.GetHost()
		// if err != nil {
		// 	logger.Fatal(err)
		// }

		// // PersistantState
		// persistantState, err := lib.GetPersistantState(hst)
		// if err != nil {
		// 	logger.Fatal(err)
		// }

		// // Load saved state
		// savedHostState, err := state.LoadHostState(ctx, persistantState)
		// if err != nil {
		// 	logger.Fatal(err)
		// }

		// // Plan
		// plan, err := resource.NewActionPlanFromHostState(
		// 	ctx, hst, *savedHostState, resource.Bundles{}, resource.ActionDestroy,
		// )
		// if err != nil {
		// 	logger.Fatal(err)
		// }
		// plan.Print(ctx)

		// // Execute plan
		// newHostState, err := plan.Execute(ctx, hst)
		// if err != nil {
		// 	logger.Fatal(err)
		// }

		// // Save state
		// if err := state.SaveHostState(ctx, newHostState, persistantState); err != nil {
		// 	logger.Fatal(err)
		// }

		// // Success
		// logger.Info("🎆 Success")
	},
}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)
}
