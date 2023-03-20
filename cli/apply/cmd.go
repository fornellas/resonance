package apply

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/cli/lib"
	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/state"
)

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

var Cmd = &cobra.Command{
	Use:   "apply [flags] resources_root",
	Short: "Applies configuration to a host.",
	Long:  "Loads all resoures from .yaml files at resources_root, the previous state, craft a plan and applies required changes to given host.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		root := args[0]

		logger := log.GetLogger(ctx)

		// Host
		hst, err := lib.GetHost()
		if err != nil {
			logger.Fatal(err)
		}

		// PersistantState
		persistantState, err := lib.GetPersistantState(hst)
		if err != nil {
			logger.Fatal(err)
		}

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

		// Rollback NewRollbackBundle
		rollbackBundle := resource.NewRollbackBundle(
			newBundle, previousBundle, typeNameStateMap, resource.ActionConfigure,
		)

		// TODO save rollback bundle

		// Plan
		plan, err := resource.NewPlan(
			ctx, hst, newBundle, previousBundle, typeNameStateMap, resource.ActionConfigure,
		)
		if err != nil {
			logger.Fatal(err)
		}
		plan.Print(ctx, hst)

		// Execute plan
		err = plan.Execute(ctx, hst)

		if err == nil {
			// Save plan state
			newHostState := resource.NewHostState(newBundle)
			if err := state.SaveHostState(ctx, newHostState, persistantState); err != nil {
				logger.Fatal(err)
			}

			logger.Info("🎆 Success")
		} else {
			nestedCtx := log.IndentLogger(ctx)
			nestedLogger := log.GetLogger(nestedCtx)
			nestedLogger.Error(err)
			lib.Rollback(ctx, hst, rollbackBundle)
		}
	},
}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)
}
