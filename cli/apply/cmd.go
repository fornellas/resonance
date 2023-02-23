package apply

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

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
		ctx := context.Background()

		var hst host.Host
		if localhost {
			hst = host.Local{}
		} else if hostname != "" {
			hst = host.Ssh{
				Hostname: hostname,
			}
		} else {
			logrus.Fatal(errors.New("must provide either --localhost or --hostname"))
		}

		localState := state.Local{
			Path: stateYaml,
		}
		savedStateData, err := localState.Load(ctx)
		if err != nil {
			logrus.Fatal(err)
		}

		var resourceBundles []resource.ResourceDefinitions
		for _, path := range args {
			pathResourceDefinitions, err := resource.Load(ctx, path)
			if err != nil {
				logrus.Fatal(err)
			}
			resourceBundles = append(resourceBundles, pathResourceDefinitions)
		}
		// order resources

		// read initial state of all resources

		// if saved / initial state differ
		// 	read initial inventory
		// 	apply resources that are different
		// 	destroy resources that are not present anymore
		// 	read final state of all resources
		// 	save state of all resources
		// 	read final inventory
		// 	if initial / final inventory differ
		// 		big fat warning
		// 		if more than once
		// 			fail
		// 		start over

		logrus.Fatal("TODO apply.Run")
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
		logrus.Fatal(err)
	}
}
