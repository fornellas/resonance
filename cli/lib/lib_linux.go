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

	return wrapHost(ctx, hst)
}

func AddHostFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(
		&localhost, "localhost", "", defaultLocalhost,
		"Applies configuration to the same machine running the command",
	)

	addHostFlagsCommon(cmd)
}
