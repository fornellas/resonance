package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host/types"
)

func getHostArch(context.Context) (types.BaseHost, types.Host) {
	return nil, nil
}

func addTargetFlagsArch(cmd *cobra.Command) []string {
	targetFlagNames := []string{}

	return targetFlagNames
}
