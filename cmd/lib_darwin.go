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
	var hst host.Host
	var err error

	if ssh != "" {
		hst, err = ihost.NewSshAuthority(ctx, ssh)
		if err != nil {
			return nil, err
		}
	} else if dockerContainer != "" {
		hst, err = ihost.NewDocker(ctx, dockerContainer, dockerUser)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("no target host specified: must pass either --ssh or --docker-container")
	}

	return wrapHost(ctx, hst)
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
