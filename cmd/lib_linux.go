package main

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
	ihost "github.com/fornellas/resonance/internal/host"
	storePkg "github.com/fornellas/resonance/internal/store"
)

var localhost bool
var defaultLocalhost = false

var storeHostLocalhostPath string
var defaultStoreHostLocalhostPath = "state/"

func GetHost(ctx context.Context) (host.Host, error) {
	var hst host.Host
	var err error

	if localhost {
		hst = ihost.Local{}
	} else if ssh != "" {
		hst, err = ihost.NewSshAuthority(ctx, ssh)
		if err != nil {
			return nil, err
		}
	} else if docker != "" {
		hst, err = ihost.NewDocker(ctx, docker)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("no target host specified: must pass either --localhost, --ssh or --docker-container")
	}

	return wrapHost(ctx, hst)
}

func AddHostFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(
		&localhost, "localhost", "", defaultLocalhost,
		"Applies configuration to the same machine running the command",
	)

	addHostFlagsCommon(cmd)
}

func AddStoreFlags(cmd *cobra.Command) {
	addStoreFlagsCommon(cmd)

	cmd.Flags().StringVarP(
		&storeHostLocalhostPath, "store-localhost-path", "", defaultStoreHostLocalhostPath,
		"Path on localhost where to store state",
	)
}

func GetStore(hst host.Host) storePkg.Store {
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
