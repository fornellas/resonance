package main

import (
	"context"

	"errors"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
	ihost "github.com/fornellas/resonance/internal/host"
	storePkg "github.com/fornellas/resonance/internal/store"
)

func GetHost(ctx context.Context) (host.Host, error) {
	var baseHost host.BaseHost
	var err error

	if ssh != "" {
		baseHost, err = ihost.NewSshAuthority(ctx, ssh)
		if err != nil {
			return nil, err
		}
	} else if docker != "" {
		baseHost, err = ihost.NewDocker(ctx, docker)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("no target host specified: must pass either --target-ssh or --target-docker")
	}

	if sudo {
		var err error
		baseHost, err = ihost.NewSudoWrapper(ctx, baseHost)
		if err != nil {
			return nil, err
		}
	}

	hst, err := ihost.NewAgentClientWrapper(ctx, baseHost)
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

func GetStore(hst host.Host) storePkg.Store {
	store := getStoreCommon(hst)
	if store == nil {
		panic("bug: invalid store")
	}
	return store
}
