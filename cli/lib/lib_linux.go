package lib

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
)

var localhost bool
var defaultLocalhost = false

func Reset() {
	localhost = defaultLocalhost
	resetCommon()
}

func GetHost(ctx context.Context) (host.Host, error) {
	var hst host.Host
	var err error

	if localhost {
		hst = host.Local{}
	} else if ssh != "" {
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
