package apply

import (
	"errors"

	"github.com/spf13/cobra"

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
		localState := state.Local{
			Path: stateYaml,
		}

		// Load resources
		resourceBundles, err := resource.LoadResourceBundles(ctx, args)
		if err != nil {
			logger.Fatal(err)
		}

		// Load saved state
		savedHostState, err := state.LoadHostState(ctx, localState)
		if err != nil {
			logger.Fatal(err)
		}

		// Plan
		plan, err := resource.NewPlanFromResourceBundles(ctx, hst, savedHostState, resourceBundles)
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
		if err := state.SaveHostState(ctx, newHostState, localState); err != nil {
			logger.Fatal(err)
		}

		// Success
		logger.Info("🎆 Success")
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
