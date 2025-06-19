package main

import (
	"fmt"
	"path/filepath"

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

func getStoreArch(storeType string) (storePkg.Store, string, error) {
	if storeType == "local" {
		storeLocalhostPathAbs, err := filepath.Abs(storeLocalhostPath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to get absolute path for store local path: %w", err)
		}
		return storePkg.NewHostStore(host.Local{}, storeLocalhostPathAbs), storeLocalhostPathAbs, nil
	}
	return nil, "", nil
}

func init() {
	storeNameMap["local"] = true
	resetFlagsFns = append(resetFlagsFns, func() {
		storeLocalhostPath = defaultStoreLocalhostPath
	})
}
