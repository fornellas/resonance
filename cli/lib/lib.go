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

	logger.Fatal("Failed, rollback to previously saved state successful.")
}

func readState(
	ctx context.Context,
	hst host.Host,
	newBundle resource.Bundle,
	hostState *resource.HostState,
) resource.TypeNameStateMap {
	logger := log.GetLogger(ctx)
	typeNames := newBundle.TypeNames()
	if hostState != nil {
		for _, typeName := range hostState.PreviousBundle.TypeNames() {
			duplicate := false
			for _, tn := range typeNames {
				if typeName == tn {
					duplicate = true
					break
				}
			}
			if duplicate {
				continue
			}
			typeNames = append(typeNames, typeName)
		}
	}
	typeNameStateMap, err := resource.GetTypeNameStateMap(ctx, hst, typeNames)
	if err != nil {
		logger.Fatal(err)
	}
	return typeNameStateMap
}

func Plan(
	ctx context.Context, hst host.Host, persistantState state.PersistantState, root string,
) (
	resource.Bundle, resource.Plan, resource.Bundle,
) {
	logger := log.GetLogger(ctx)

	// Load resources
	newBundle, err := resource.LoadBundle(ctx, hst, root)
	if err != nil {
		logger.Fatal(err)
	}

	// Load saved HostState
	hostState, err := state.LoadHostState(ctx, persistantState)
	if err != nil {
		logger.Fatal(err)
	}
	var previousBundle *resource.Bundle
	if hostState != nil {
		previousBundle = &hostState.PreviousBundle
	}

	// Read state
	typeNameStateMap := readState(ctx, hst, newBundle, hostState)

	// Check saved HostState
	if hostState != nil {
		isClean, err := hostState.PreviousBundle.IsClean(ctx, hst, typeNameStateMap)
		if err != nil {
			logger.Fatal(err)
		}
		if !isClean {
			logger.Fatalf(
				"Host state is not clean: this often means external agents altered the host state after previous apply. Try the 'refresh' or 'restore' commands.",
			)
		}
	}

	// Rollback Bundle
	rollbackBundle := resource.NewRollbackBundle(
		newBundle, previousBundle, typeNameStateMap, resource.ActionConfigure,
	)

	// Plan
	plan, err := resource.NewPlan(
		ctx, hst, newBundle, previousBundle, typeNameStateMap, resource.ActionConfigure,
	)
	if err != nil {
		logger.Fatal(err)
	}
	plan.Print(ctx, hst)

	return newBundle, plan, rollbackBundle
}
