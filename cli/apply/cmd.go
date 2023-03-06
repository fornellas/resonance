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
var stateFile string

var Cmd = &cobra.Command{
	Use:   "apply [flags] root_path",
	Short: "Applies configuration to a host.",
	Long:  "Loads all resoures from .yaml files at root_path, the previous state, craft a plan and applies required changes to given host.",
	Args:  cobra.ExactArgs(1),
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
			Path: stateFile,
		}

		// Load resources
		resourceBundles, err := resource.LoadBundles(ctx, args[0])
		if err != nil {
			logger.Fatal(err)
		}

		// Load saved state
		savedHostState, err := state.LoadHostState(ctx, localState)
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
		&stateFile, "state-file", "", "",
		"Path to a file to store state",
	)
	if err := Cmd.MarkFlagRequired("state-file"); err != nil {
		panic(err)
	}
}
