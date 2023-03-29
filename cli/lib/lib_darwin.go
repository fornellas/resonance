package lib

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
)

func Reset() {
	resetCommon()
}

func GetHost(ctx context.Context) (host.Host, error) {
	var hst host.Host
	var err error

	hst, err = host.NewSshAuthority(ctx, hostname)
	if err != nil {
		return nil, err
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
	addHostFlagsCommon(cmd)
}
