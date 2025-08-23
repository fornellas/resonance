package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host/types"
	storePkg "github.com/fornellas/resonance/store"
)

var storeNameMap = map[string]bool{
	"remote": true,
}

type StoreValue struct {
	name string
}

func NewStoreValue() *StoreValue {
	storeValue := &StoreValue{}
	storeValue.Reset()
	return storeValue
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

func (s *StoreValue) Reset() {
	s.name = "remote"
}

func (s *StoreValue) Type() string {
	names := []string{}
	for name := range storeNameMap {
		names = append(names, name)
	}
	return fmt.Sprintf("[%s]", strings.Join(names, "|"))
}

var storeValue = NewStoreValue()

var storeHostPath string
var defaultStoreHostPath = "/var/lib/resonance"

func AddStoreFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().VarP(storeValue, "store", "", "Where to store state information")

	cmd.Flags().StringVarP(
		&storeHostPath, "store-remote-path", "", defaultStoreHostPath,
		"Path on remote host where to store state",
	)

	addStoreFlagsArch(cmd)
}

func GetStore(host types.Host) (storePkg.Store, string, error) {
	store, config, err := getStoreArch(storeValue.String())
	if err != nil {
		return nil, "", err
	}
	if store != nil {
		return storePkg.NewLoggingWrapper(store), config, nil
	}

	switch storeValue.String() {
	case "remote":
		return storePkg.NewLoggingWrapper(
			storePkg.NewHostStore(host, storeHostPath),
		), storeHostPath, nil
	default:
		panic("bug: unexpected store value")
	}
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		storeValue.Reset()
		storeHostPath = defaultStoreHostPath
	})
}
