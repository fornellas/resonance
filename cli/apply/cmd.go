package apply

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/openconfig/goyang/pkg/indent"

	"github.com/fornellas/resonance/log"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/state"
)

var localhost bool
var hostname string
var stateYaml string

var Cmd = &cobra.Command{
	Use:   "apply [flags] yaml...",
	Short: "Applies configuration to a host.",
	Long:  "Applies configuration at yaml files to a host.\n\nA target host must be specified with either --localhost or --hostname.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		logger := log.GetLogger(ctx)

		// Host
		var hst host.Host
		if localhost {
			hst = host.Local{}
		} else if hostname != "" {
			hst = host.Ssh{
				Hostname: hostname,
			}
		} else {
			logger.Fatal(errors.New("must provide either --localhost or --hostname"))
		}

		// Local state
		logger.Info("Loading saved host state")
		localState := state.Local{
			Path: stateYaml,
		}

		// Load resources
		logger.Info("Loading resources")
		resourceBundles := resource.LoadResourceBundles(ctx, args)

		// Get current host state
		// logger.Info("Getting current host state")
		// currentHostState, err := resourceBundles.GetHostState(ctx, hst)
		// if err != nil {
		// 	logger.Fatal(err)
		// }
		// currentHostStateStr, err := currentHostState.String()
		// if err != nil {
		// 	logger.Fatal(err)
		// }
		// logger.Debugf("currentHostStateStr:\n%v", indent.String("  ", currentHostStateStr))
		// TODO if currentHostState != savedHostState
		// - something changed the state outside, do `resonance state push`
		// - How to deal with partial resource parameters (eg: APTPackage without Version)
		// - should host state also save resource bundles?

		// Load saved state
		// savedHostState, err := localState.Load(ctx)
		// if err != nil && !errors.Is(err, fs.ErrNotExist) {
		// 	logger.Fatal(err)
		// }
		// savedHostStateStr, err := savedHostState.String()
		// if err != nil {
		// 	logger.Fatal(err)
		// }
		// logger.Debugf("savedHostState:\n%s", indent.String("  ", savedHostStateStr))

		// Get desired host state
		// logger.Info("Calculating desired host state")
		// desiredHostState, err := resourceBundles.GetDesiredHostState()
		// if err != nil {
		// 	logger.Fatal(err)
		// }
		// desiredHostStateStr, err := desiredHostState.String()
		// if err != nil {
		// 	logger.Fatal(err)
		// }
		// logger.Debugf("desiredHostStateStr:\n%s", indent.String("  ", desiredHostStateStr))

		// Plan
		logger.Info("Planning changes")
		plan, err := resourceBundles.GetPlan(ctx, hst, localState)
		if err != nil {
			logger.Fatal(err)
		}
		logger.Debugf("plan:\n%v", indent.String("  ", plan.Graphviz()))

		// Applying changes
		logger.Info("Applying changes")
		if err := plan.Apply(ctx, hst); err != nil {
			logger.Fatal(err)
		}

		// read final state of all resources
		// save state of all resources

		logger.Fatal("TODO resonance apply")
	},
}

func init() {
	Cmd.Flags().BoolVarP(
		&localhost, "localhost", "", false,
		"Applies configuration to the same machine running the command",
	)

	Cmd.Flags().StringVarP(
		&hostname, "hostname", "", "",
		"Applies configuration to given hostname using SSH",
	)

	Cmd.Flags().StringVarP(
		&stateYaml, "state-yaml", "", "",
		"Path to a yaml file to store state",
	)
	if err := Cmd.MarkFlagRequired("state-yaml"); err != nil {
		panic(err)
	}
}
