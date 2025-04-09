package main

import (
	"github.com/fornellas/resonance/host"
	storePkg "github.com/fornellas/resonance/store"
)

func getLocalStore() storePkg.Store {
	return storePkg.NewHostStore(host.Local{}, storeLocalhostPath)
}
