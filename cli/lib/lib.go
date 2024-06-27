package lib

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
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
	dockerContainer = defaultDockerContainer
	dockerUser = defaultDockerUser
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

func AddPersistantStateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&stateRoot, "state-root", "", "/var/lib/resonance",
		"Root path at host where to save host state to.",
	)
}
