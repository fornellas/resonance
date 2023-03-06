package lib

import (
	"errors"
	"path/filepath"

	"github.com/adrg/xdg"
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

var stateRoot string

func GetPersistantState(hst host.Host) (state.PersistantState, error) {
	return state.NewLocal(stateRoot, hst), nil
}

func AddPersistantStateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&stateRoot, "state-root", "", filepath.Join(xdg.StateHome, "resonance", "state"),
		"Root path where to save host state to.",
	)
}
