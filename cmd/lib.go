package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
	ihost "github.com/fornellas/resonance/internal/host"
	storePkg "github.com/fornellas/resonance/internal/store"
)

// This is to be used in place of os.Exit() to aid writing test assertions on exit code.
var Exit func(int) = func(code int) { os.Exit(code) }

var ssh string
var defaultSsh = ""

var docker string
var defaultDocker = ""

var sudo bool
var defaultSudo = false

var disableAgent bool
var defaultDisableAgent = false

var storeValue = NewStoreValue()

var storeHostTargetPath string
var defaultStoreHostTargetPath = "/var/lib/resonance"

func wrapHost(ctx context.Context, hst host.Host) (host.Host, error) {
	var err error
	if sudo {
		hst, err = ihost.NewSudo(ctx, hst)
		if err != nil {
			return nil, err
		}
	}

	if !disableAgent && ssh != "" {
		var err error
		hst, err = ihost.NewAgent(ctx, hst)
		if err != nil {
			return nil, err
		}
	}

	return hst, nil
}

func addHostFlagsCommon(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&ssh, "ssh", "", defaultSsh,
		"Applies configuration to given hostname using SSH in the format: [<user>[;fingerprint=<host-key fingerprint>]@]<host>[:<port>]",
	)

	cmd.Flags().StringVarP(
		&docker, "docker", "", defaultDocker,
		"Applies configuration to given Docker container name \n"+
			"Use given format 'USER@CONTAINER_ID'",
	)

	cmd.Flags().BoolVarP(
		&sudo, "sudo", "", defaultSudo,
		"Use sudo when interacting with host",
	)

	cmd.Flags().BoolVarP(
		&disableAgent, "disable-agent", "", defaultDisableAgent,
		"Disables copying temporary a small agent to remote hosts. This can make things very slow, as without the agent, iteraction require running multiple commands. The only (unusual) use case for this is when the host architecture is not supported by the agent.",
	)
}

func addStoreFlagsCommon(cmd *cobra.Command) {
	cmd.PersistentFlags().VarP(storeValue, "store", "", "Where to store state information")

	cmd.Flags().StringVarP(
		&storeHostTargetPath, "store-target-path", "", defaultStoreHostTargetPath,
		"Path on target host where to store state",
	)
}

func getStoreCommon(hst host.Host) storePkg.Store {
	var storePath string
	switch storeValue.String() {
	case "target":
		storePath = storeHostTargetPath
	default:
		return nil
	}
	return storeValue.GetStore(hst, storePath)
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		ssh = defaultSsh
		docker = defaultDocker
		sudo = defaultSudo
		disableAgent = defaultDisableAgent
		storeHostTargetPath = defaultStoreHostTargetPath
	})
}
