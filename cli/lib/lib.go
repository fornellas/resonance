package lib

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/state"
)

var localhost bool
var hostname string

func GetHost() (host.Host, error) {
	var hst host.Host
	if localhost {
		hst = host.Local{}
	} else if hostname != "" {
		hst = host.Ssh{
			Hostname: hostname,
		}
	} else {
		return nil, errors.New("must provide either --localhost or --hostname")
	}
	return hst, nil
}

func AddHostFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(
		&localhost, "localhost", "", false,
		"Applies configuration to the same machine running the command",
	)

	cmd.Flags().StringVarP(
		&hostname, "hostname", "", "",
		"Applies configuration to given hostname using SSH",
	)
}

var stateFile string

func GetPersistantState() (state.PersistantState, error) {
	return state.Local{
		Path: stateFile,
	}, nil
}

func AddPersistantStateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&stateFile, "state-file", "", "",
		"Path to a file to store state",
	)
	if err := cmd.MarkFlagRequired("state-file"); err != nil {
		panic(err)
	}
}
