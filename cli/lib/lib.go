package lib

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"
	"github.com/fornellas/resonance/state"
)

var ssh string
var defaultSsh = ""

var dockerContainer string
var defaultDockerContainer = ""
var dockerUser string
var defaultDockerUser = "0:0"

var sudo bool
var defaultSudo = false

var disableAgent bool
var defaultDisableAgent = false

func resetCommon() {
	ssh = defaultSsh
	sudo = defaultSudo
	disableAgent = defaultDisableAgent
}

func wrapHost(ctx context.Context, hst host.Host) (host.Host, error) {
	var err error
	if sudo {
		hst, err = host.NewSudo(ctx, hst)
		if err != nil {
			return nil, err
		}
	}

	if !disableAgent && ssh != "" {
		var err error
		hst, err = host.NewAgent(ctx, hst)
		if err != nil {
			return nil, err
		}
	}

	return hst, nil
}

func addHostFlagsCommon(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&ssh, "ssh", "", defaultSsh,
		"Applies configuration to given hostname using SSH in the format: [<user>[;fingerprint=<host-key fingerprint>]@]<host>[:<port>]",
	)

	cmd.Flags().StringVarP(
		&dockerContainer, "docker-container", "", defaultDockerContainer,
		"Applies configuration to given Docker container name",
	)

	cmd.Flags().StringVarP(
		&dockerUser, "docker-user", "", defaultDockerUser,
		"Use given user/group in the format '<name|uid>[:<group|gid>]'",
	)

	cmd.Flags().BoolVarP(
		&sudo, "sudo", "", defaultSudo,
		"Use sudo when interacting with host",
	)

	cmd.Flags().BoolVarP(
		&disableAgent, "disable-agent", "", defaultDisableAgent,
		"Disables copying temporary a small agent to remote hosts. This can make things very slow, as without the agent, iteraction require running multiple commands. The only (unusual) use case for this is when the host architecture is not supported by the agent.",
	)
}

var stateRoot string

func GetPersistantState(hst host.Host) (state.PersistantState, error) {
	return state.NewHost(stateRoot, hst), nil
}

func AddPersistantStateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&stateRoot, "state-root", "", "/var/lib/resonance",
		"Root path at host where to save host state to.",
	)
}

func Rollback(ctx context.Context, hst host.Host, rollbackBundle resource.Bundle) error {
	logger := log.GetLogger(ctx)
	logger.Warn("Attempting rollback")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	// Read current state
	typeNameStateMap, err := resource.GetTypeNameStateMap(nestedCtx, hst, rollbackBundle.TypeNames(), true)
	if err != nil {
		return err
	}

	// Rollback Plan
	rollbackPlan, err := resource.NewPlan(
		nestedCtx, rollbackBundle, nil, typeNameStateMap, resource.ActionConfigure,
	)
	if err != nil {
		return err
	}
	rollbackPlan.Print(nestedCtx, hst)

	// Execute plan
	err = rollbackPlan.Execute(nestedCtx, hst)
	if err != nil {
		return err
	}
	nestedLogger.Info("👌 Rollback successful.")
	return nil
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
	typeNameStateMap, err := resource.GetTypeNameStateMap(ctx, hst, typeNames, true)
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
		dirtyMsg, err := hostState.IsClean(ctx, hst, typeNameStateMap)
		if err != nil {
			logger.Fatal(err)
		}
		if dirtyMsg != "" {
			logger.Fatal(dirtyMsg)
		}
	}

	// Rollback Bundle
	rollbackBundle := resource.NewRollbackBundle(
		newBundle, previousBundle, typeNameStateMap, resource.ActionConfigure,
	)

	// Plan
	plan, err := resource.NewPlan(
		ctx, newBundle, previousBundle, typeNameStateMap, resource.ActionConfigure,
	)
	if err != nil {
		logger.Fatal(err)
	}
	plan.Print(ctx, hst)

	return newBundle, plan, rollbackBundle
}
