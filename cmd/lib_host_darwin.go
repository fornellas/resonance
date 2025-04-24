package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host/types"
)

func getHostArch(context.Context) (types.BaseHost, types.Host) {
	return nil, nil
}

func addHostFlagsArch(_ *cobra.Command) []string {
	hostFlagNames := []string{}

	return hostFlagNames
}
