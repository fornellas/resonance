package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host/types"
)

func getBaseHostArch(context.Context) types.BaseHost {
	return nil
}

func addHostFlagsArch(_ *cobra.Command) []string {
	hostFlagNames := []string{}

	return hostFlagNames
}
