package main

import (
	"context"

	"github.com/spf13/cobra"

	hostPkg "github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
)

var localhost bool
var defaultLocalhost = false

func getBaseHostArch(context.Context) types.BaseHost {
	if localhost {
		return hostPkg.Local{}
	}
	return nil
}

func addHostFlagsArch(cmd *cobra.Command) []string {
	hostFlagNames := []string{}

	cmd.Flags().BoolVarP(
		&localhost, "host-local", "1", defaultLocalhost,
		"Applies configuration to the same machine running the command",
	)
	hostFlagNames = append(hostFlagNames, "host-local")

	return hostFlagNames
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		localhost = defaultLocalhost
	})
}
