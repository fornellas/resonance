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
	} else if hostname != "" {
		hst, err = host.NewSshAuthority(ctx, hostname)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("must provide either --localhost or --hostname")
	}

	if sudo {
		hst, err = host.NewSudo(ctx, hst)
		if err != nil {
			return nil, err
		}
	}

	if !disableAgent && hostname != "" {
		var err error
		hst, err = host.NewAgent(ctx, hst)
		if err != nil {
			return nil, err
		}
	}

	return hst, nil
}

func AddHostFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(
		&localhost, "localhost", "", defaultLocalhost,
		"Applies configuration to the same machine running the command",
	)

	addHostFlagsCommon(cmd)
}
