package apply

import (
	"context"
	"errors"
	"io/fs"
	"reflect"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/state"
)

var localhost bool
var hostname string
var stateYaml string

func getHost() host.Host {
	if localhost {
		return host.Local{}
	} else if hostname != "" {
		return host.Ssh{
			Hostname: hostname,
		}
	} else {
		logrus.Fatal(errors.New("must provide either --localhost or --hostname"))
	}
	return nil
}

func LoadResourceDefinitions(ctx context.Context, paths []string) resource.ResourceDefinitions {
	resourceDefinitions := resource.ResourceDefinitions{}
	for _, path := range paths {
		pathResourceDefinitions, err := resource.LoadResourceDefinitions(ctx, path)
		if err != nil {
			logrus.Fatal(err)
		}
		resourceDefinitions = append(resourceDefinitions, pathResourceDefinitions...)
	}
	return resourceDefinitions
}

var Cmd = &cobra.Command{
	Use:   "apply [flags] yaml...",
	Short: "Applies configuration to a host.",
	Long:  "Applies configuration at yaml files to a host.\n\nA target host must be specified with either --localhost or --hostname.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		hst := getHost()

		localState := state.Local{
			Path: stateYaml,
		}
		savedHostState, err := localState.Load(ctx)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			logrus.Fatal(err)
		}

		resourceDefinitions := LoadResourceDefinitions(ctx, args)

		currentHostState, err := resourceDefinitions.ReadState(ctx, hst)
		if err != nil {
			logrus.Fatal(err)
		}

		if reflect.DeepEqual(savedHostState, currentHostState) {
			logrus.Info("Nothing to do")
			return
		}

		// merge resources
		// Define execution order
		// read initial inventory: if different from saved, it means state is busted
		// apply resources that are different
		// destroy resources that are not present anymore
		// read final state of all resources
		// save state of all resources
		// read final inventory
		// if initial / final inventory differ
		// 	big fat warning
		// 	if more than once
		// 		fail
		// 	start over

		logrus.Fatal("TODO resonance apply")
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
