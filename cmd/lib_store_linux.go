package main

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
	storePkg "github.com/fornellas/resonance/store"
)

var storeLocalhostPath string
var defaultStoreLocalhostPath = "state/"

func addStoreFlagsArch(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&storeLocalhostPath, "store-local-path", "", defaultStoreLocalhostPath,
		"Path on localhost where to store state",
	)
}

func getStoreArch(storeType string) (storePkg.Store, string) {
	if storeType == "local" {
		return storePkg.NewHostStore(host.Local{}, storeLocalhostPath), storeLocalhostPath
	}
	return nil, ""
}

func init() {
	storeNameMap["local"] = true
	resetFlagsFns = append(resetFlagsFns, func() {
		storeLocalhostPath = defaultStoreLocalhostPath
	})
}
