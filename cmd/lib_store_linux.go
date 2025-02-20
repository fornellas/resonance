package main

import (
	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
	storePkg "github.com/fornellas/resonance/store"
)

func getStoreArch(hst types.Host) storePkg.Store {
	if storeValue.String() == "localhost" {
		return storePkg.NewHostStore(host.Local{}, storeLocalhostPath)
	}
	return nil
}
