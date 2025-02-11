package main

import (
	"context"

	"errors"

	"github.com/spf13/cobra"

	hostPkg "github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
	storePkg "github.com/fornellas/resonance/store"
)

func GetHost(ctx context.Context) (types.Host, error) {
	var baseHost types.BaseHost
	var err error

	if ssh != "" {
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
		return nil, errors.New("no target host specified: must pass either --target-ssh or --target-docker")
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
	addHostFlagsCommon(cmd)
}

func AddStoreFlags(cmd *cobra.Command) {
	addStoreFlagsCommon(cmd)
}

func GetStore(hst types.Host) storePkg.Store {
	store := getStoreCommon(hst)
	if store == nil {
		panic("bug: invalid store")
	}
	return store
}
