package main

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
	ihost "github.com/fornellas/resonance/internal/host"
	storePkg "github.com/fornellas/resonance/internal/store"
	"github.com/fornellas/resonance/log"
)

// This is to be used in place of os.Exit() to aid writing test assertions on exit code.
var Exit func(int) = func(code int) { os.Exit(code) }

var ssh string
var defaultSsh = ""

var docker string
var defaultDocker = ""

var sudo bool
var defaultSudo = false

var storeValue = NewStoreValue()

var storeHostTargetPath string
var defaultStoreHostTargetPath = "/var/lib/resonance"

func wrapHost(ctx context.Context, hst host.Host) (host.Host, error) {
	logger := log.MustLogger(ctx)

	if sudo {
		var err error
		hst, err = ihost.NewSudoWrapper(ctx, hst)
		if err != nil {
			return nil, err
		}
	}

	if sudo || hst.Type() != "localhost" {
		var err error
		hst, err = ihost.NewAgentClientWrapper(ctx, hst)
		if err != nil {
			if errors.Is(err, ihost.ErrAgentUnsupportedOsArch) {
				logger.Warn(
					"Agent has no support for target, expect things to run *really* slow",
					"err", err,
				)
			} else {
				return nil, err
			}
		}
	}

	return hst, nil
}

func addHostFlagsCommon(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&ssh, "target-ssh", "s", defaultSsh,
		"Applies configuration to given hostname using SSH in the format: [<user>[;fingerprint=<host-key fingerprint>]@]<host>[:<port>]",
	)

	cmd.Flags().StringVarP(
		&docker, "target-docker", "d", defaultDocker,
		"Applies configuration to given Docker container name \n"+
			"Use given format '[<name|uid>[:<group|gid>]@]<image>'",
	)

	cmd.Flags().BoolVarP(
		&sudo, "target-sudo", "r", defaultSudo,
		"Use sudo to gain root privileges",
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
		storeHostTargetPath = defaultStoreHostTargetPath
		sudo = defaultSudo
	})
}
