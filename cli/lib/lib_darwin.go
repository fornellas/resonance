package lib

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
)

func Reset() {
	resetCommon()
}

func GetHost(ctx context.Context) (host.Host, error) {
	var hst host.Host
	var err error

	if ssh != "" {
		hst, err = host.NewSshAuthority(ctx, ssh)
		if err != nil {
			return nil, err
		}
	} else if dockerContainer != "" {
		hst, err = host.NewDocker(ctx, dockerContainer, dockerUser)
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
