package lib

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"
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

func Rollback(ctx context.Context, hst host.Host, rollbackBundle resource.Bundle) {
	logger := log.GetLogger(ctx)
	logger.Warn("Attempting rollback")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	// Read current state
	typeNameStateMap, err := resource.GetTypeNameStateMap(nestedCtx, hst, rollbackBundle.TypeNames())
	if err != nil {
		logger.Fatal(err)
	}

	// Rollback Plan
	rollbackPlan, err := resource.NewPlan(
		nestedCtx, hst, rollbackBundle, nil, typeNameStateMap, resource.ActionConfigure,
	)
	if err != nil {
		logger.Fatal(err)
	}
	rollbackPlan.Print(nestedCtx, hst)

	// Execute plan
	err = rollbackPlan.Execute(nestedCtx, hst)
	if err != nil {
		nestedLogger.Error(err)
		logger.Fatal("Rollback failed! You may try the restore command and / or fix things manually.")
	}
	nestedLogger.Info("👌 Rollback successful.")

	// TODO save state without rollback

	logger.Fatal("Failed to apply, rollback to previously saved state successful.")
}
