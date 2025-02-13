package main

import (
	"context"

	"github.com/spf13/cobra"

	hostPkg "github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
	storePkg "github.com/fornellas/resonance/store"
)

var localhost bool
var defaultLocalhost = false

var storeHostLocalhostPath string
var defaultStoreHostLocalhostPath = "state/"

func GetHost(ctx context.Context) (types.Host, error) {
	var baseHost types.BaseHost
	var err error

	if localhost {
		if sudo {
			baseHost = hostPkg.Local{}
		} else {
			return hostPkg.Local{}, nil
		}
	} else if ssh != "" {
		baseHost, err = hostPkg.NewSshAuthority(ctx, ssh, hostPkg.SshClientConfig{
			RekeyThreshold:    sshRekeyThreshold,
			KeyExchanges:      sshKeyExchanges,
			Ciphers:           sshCiphers,
			MACs:              sshMACs,
			HostKeyAlgorithms: sshHostKeyAlgorithms,
			Timeout:           sshTcpConnectTimeout,
		})
		if err != nil {
			return nil, err
		}
	} else if docker != "" {
		baseHost, err = hostPkg.NewDocker(ctx, docker)
		if err != nil {
			return nil, err
		}
	} else {
		panic("bug: no target set")
	}

	if sudo {
		var err error
		baseHost, err = hostPkg.NewSudoWrapper(ctx, baseHost)
		if err != nil {
			return nil, err
		}
	}

	hst, err := hostPkg.NewAgentClientWrapper(ctx, baseHost)
	if err != nil {
		return nil, err
	}

	return hst, nil
}

func AddHostFlags(cmd *cobra.Command) {
	targetFlagNames := addCommonTargetFlags(cmd)

	cmd.Flags().BoolVarP(
		&localhost, "target-localhost", "1", defaultLocalhost,
		"Applies configuration to the same machine running the command",
	)
	targetFlagNames = append(targetFlagNames, "target-localhost")

	cmd.MarkFlagsMutuallyExclusive(targetFlagNames...)
	cmd.MarkFlagsOneRequired(targetFlagNames...)
}

func AddStoreFlags(cmd *cobra.Command) {
	addStoreFlagsCommon(cmd)

	cmd.Flags().StringVarP(
		&storeHostLocalhostPath, "store-localhost-path", "", defaultStoreHostLocalhostPath,
		"Path on localhost where to store state",
	)
}

func GetStore(hst types.Host) storePkg.Store {
	store := getStoreCommon(hst)
	if store != nil {
		return store
	}

	var storePath string
	switch storeValue.String() {
	case "localhost":
		storePath = storeHostLocalhostPath
	default:
		panic("bug: invalid store")
	}
	return storeValue.GetStore(hst, storePath)
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		localhost = defaultLocalhost
		storeHostLocalhostPath = defaultStoreHostLocalhostPath
	})
}
