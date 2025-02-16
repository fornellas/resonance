package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
	storePkg "github.com/fornellas/resonance/store"
)

var storeNameMap = map[string]bool{
	"target":    true,
	"localhost": true,
}

type StoreValue struct {
	name string
}

func NewStoreValue() *StoreValue {
	return &StoreValue{
		name: "target",
	}
}

func (s *StoreValue) String() string {
	return s.name
}

func (s *StoreValue) Set(value string) error {
	if _, ok := storeNameMap[value]; !ok {
		return fmt.Errorf("invalid store name '%s', valid options are %s", value, s.Type())
	}
	s.name = value
	return nil
}

func (s *StoreValue) Type() string {
	names := []string{}
	for name := range storeNameMap {
		names = append(names, name)
	}
	return fmt.Sprintf("[%s]", strings.Join(names, "|"))
}

var storeValue = NewStoreValue()

var storeTargetPath string
var defaultStoreTargetPath = "/var/lib/resonance"

var storeLocalhostPath string
var defaultStoreLocalhostPath = "state/"

func AddStoreFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().VarP(storeValue, "store", "", "Where to store state information")

	cmd.Flags().StringVarP(
		&storeTargetPath, "store-target-path", "", defaultStoreTargetPath,
		"Path on target host where to store state",
	)

	cmd.Flags().StringVarP(
		&storeLocalhostPath, "store-localhost-path", "", defaultStoreLocalhostPath,
		"Path on localhost where to store state",
	)
}

func GetStore(hst types.Host) storePkg.Store {
	switch storeValue.String() {
	case "target":
		return storePkg.NewHostStore(hst, storeTargetPath)
	case "localhost":
		return storePkg.NewHostStore(host.Local{}, storeLocalhostPath)
	default:
		panic("bug: unexpected store value")
	}
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		storeValue = NewStoreValue()
		storeTargetPath = defaultStoreTargetPath
		storeLocalhostPath = defaultStoreLocalhostPath
	})
}
