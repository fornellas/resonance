package main

import (
	"context"
	"os"

	"fmt"
	"strings"

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

var options string
var defaultOptions = ""

var storeValue = NewStoreValue()

var storeHostTargetPath string
var defaultStoreHostTargetPath = "/var/lib/resonance"

func wrapHost(ctx context.Context, hst host.Host) (host.Host, error) {
	var err error

	optionsMap := map[string]bool{
		"sudo":          false,
		"disable-agent": false,
	}
	if options != "" {
		for _, o := range strings.Split(options, ",") {
			if _, ok := optionsMap[o]; !ok {
				return nil, fmt.Errorf("invalid option: %s", o)
			}
			optionsMap[o] = true
		}

		if optionsMap["sudo"] {
			hst, err = ihost.NewSudo(ctx, hst)
			if err != nil {
				return nil, err
			}
		}

		if hst.Type() != "localhost" && !optionsMap["disable-agent"] {
			hst, err = ihost.NewAgent(ctx, hst)
			if err != nil {
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
	cmd.Flags().StringVarP(
		&options, "target-options", "o", defaultDocker,
		"Comma separated list of target options: \n"+
			"	\"sudo\", to run as root via sudo; \n"+
			"	\"disable-agent\", disable ephemeral agent usage (MUCH slower, only use for CPU architectures where there's no agent support)",
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
		options = defaultOptions
		storeHostTargetPath = defaultStoreHostTargetPath
	})
}
