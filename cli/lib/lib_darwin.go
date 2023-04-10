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

	return wrapHost(ctx, hst)
}

func AddHostFlags(cmd *cobra.Command) {
	addHostFlagsCommon(cmd)
}
