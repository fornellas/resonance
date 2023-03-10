package plan

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/cli/lib"
)

var graphvizPath string

var Cmd = &cobra.Command{
	Use:   "plan [flags] resources_root",
	Short: "Plan required configuration to a host.",
	Long:  "Loads all resoures from .yaml files at resources_root, the previous state, craft a plan and prints it out without applying any changes.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		panic("plan")
		// ctx := cmd.Context()

		// logger := log.GetLogger(ctx)
		// nestedCtx := log.IndentLogger(ctx)
		// nestedLogger := log.GetLogger(nestedCtx)

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

		// // Load resources
		// resourceBundles, err := resource.LoadBundles(ctx, args[0])
		// if err != nil {
		// 	logger.Fatal(err)
		// }

		// // Load saved state
		// savedHostState, err := state.LoadHostState(ctx, persistantState)
		// if err != nil {
		// 	logger.Fatal(err)
		// }

		// // Plan
		// plan, err := resource.NewPlanFromBundles(ctx, hst, savedHostState, resourceBundles)
		// if err != nil {
		// 	logger.Fatal(err)
		// }

		// // Print
		// if graphvizPath != "" {
		// 	graphviz := plan.Graphviz()
		// 	logger.Info("📝 Plan")

		// 	nestedLogger.Infof(
		// 		"🔗 http://magjac.com/graphviz-visual-editor/?dot=%s", url.QueryEscape(graphviz),
		// 	)

		// 	if graphvizPath == "-" {
		// 		fmt.Printf("%s", graphviz)
		// 	} else {
		// 		nestedLogger.Infof("📝 Writing plan to %s", graphvizPath)
		// 		if err := os.WriteFile(graphvizPath, []byte(graphviz), 0640); err != nil {
		// 			logger.Fatal(err)
		// 		}
		// 	}
		// } else {
		// 	plan.Print(ctx)
		// }
	},
}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)

	Cmd.Flags().StringVarP(
		&graphvizPath, "graphviz-path", "", "",
		`Generate a Graphviz DOT graph containing the plan and write it to given file. If "-" is given, then it is printed to stdout.`,
	)
}
