package main

import (
	"github.com/spf13/cobra"

	storePkg "github.com/fornellas/resonance/store"
)

func addStoreFlagsArch(cmd *cobra.Command) {}

func getStoreArch(_ string) (storePkg.Store, string) {
	return nil, ""
}
