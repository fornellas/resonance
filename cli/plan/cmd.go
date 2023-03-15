package plan

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/cli/lib"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/state"
)

var graphvizPath string

var Cmd = &cobra.Command{
	Use:   "plan [flags] resources_root",
	Short: "Plan required configuration to a host.",
	Long:  "Loads all resoures from .yaml files at resources_root, the previous state, craft a plan and prints it out without applying any changes.",
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
		persistantState, err := lib.GetPersistantState(hst)
		if err != nil {
			logger.Fatal(err)
		}

		// Load resources
		root := args[0]
		bundle, err := resource.LoadBundle(ctx, root)
		if err != nil {
			logger.Fatal(err)
		}

		// Load saved HostState
		savedHostState, err := state.LoadHostState(ctx, persistantState)
		if err != nil {
			logger.Fatal(err)
		}

		// Read current state
		var initialResources resource.Resources
		if savedHostState != nil {
			initialResources = savedHostState.Resources
		}
		initialResources = initialResources.AppendIfNotPresent(bundle.Resources())
		initialResourcesState, err := resource.NewResourcesState(ctx, hst, initialResources)
		if err != nil {
			logger.Fatal(err)
		}

		// Check saved HostState
		if savedHostState != nil {
			if err := savedHostState.Check(ctx, hst, initialResourcesState); err != nil {
				logger.Fatal(err)
			}
		}

		// Plan
		plan, err := resource.NewPlanFromSavedStateAndBundle(
			ctx, hst, bundle, savedHostState, initialResourcesState, resource.ActionNone,
		)
		if err != nil {
			logger.Fatal(err)
		}
		plan.Print(ctx)

		// Print
		if graphvizPath != "" {
			nestedCtx := log.IndentLogger(ctx)
			nestedLogger := log.GetLogger(nestedCtx)
			graphviz := plan.Graphviz()
			logger.Info("📝 Plan")

			nestedLogger.Infof(
				"🔗 http://magjac.com/graphviz-visual-editor/?dot=%s", url.QueryEscape(graphviz),
			)

			if graphvizPath == "-" {
				fmt.Printf("%s", graphviz)
			} else {
				nestedLogger.Infof("📝 Writing plan to %s", graphvizPath)
				if err := os.WriteFile(graphvizPath, []byte(graphviz), 0640); err != nil {
					logger.Fatal(err)
				}
			}
		} else {
			plan.Print(ctx)
		}
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
