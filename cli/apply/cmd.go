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
		resourceBundles := resource.LoadResourceBundles(ctx, args)

		// Load saved state
		savedResourceBundle, err := resource.LoadSavedState(ctx, localState)
		if err != nil {
			logger.Fatal(err)
		}

		// Plan
		plan, err := resource.NewPlan(ctx, hst, savedResourceBundle, resourceBundles)
		if err != nil {
			logger.Fatal(err)
		}
		plan.Print(ctx)

		// Execute plan
		if err := plan.Execute(ctx, hst); err != nil {
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
