package main

import (
	"context"

	"github.com/spf13/cobra"

	hostPkg "github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
)

var localhost bool
var defaultLocalhost = false

func getHostArch(context.Context) (types.BaseHost, types.Host) {
	if localhost {
		if sudo {
			return hostPkg.Local{}, nil
		} else {
			return nil, hostPkg.Local{}
		}
	}
	return nil, nil
}

func addTargetFlagsArch(cmd *cobra.Command) []string {
	targetFlagNames := []string{}

	cmd.Flags().BoolVarP(
		&localhost, "target-localhost", "1", defaultLocalhost,
		"Applies configuration to the same machine running the command",
	)
	targetFlagNames = append(targetFlagNames, "target-localhost")

	return targetFlagNames
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		localhost = defaultLocalhost
	})
}
