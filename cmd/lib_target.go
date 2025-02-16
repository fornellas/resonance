package main

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	hostPkg "github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
)

var ssh string
var defaultSsh = ""

var sshRekeyThreshold uint64
var defaultSshRekeyThreshold uint64 = 0

var sshKeyExchanges []string
var defaultSshKeyExchanges = []string{}

var sshCiphers []string
var defaultSshCiphers = []string{}

var sshMACs []string
var defaultSshMACs = []string{}

var sshHostKeyAlgorithms []string
var defaultSshHostKeyAlgorithms = []string{}

var sshTcpConnectTimeout time.Duration
var defaultSshTcpConnectTimeout = time.Second * 30

var docker string
var defaultDocker = ""

var sudo bool
var defaultSudo = false

func AddTargetFlags(cmd *cobra.Command) {
	targetFlagNames := []string{}

	// Ssh
	cmd.Flags().StringVarP(
		&ssh, "target-ssh", "s", defaultSsh,
		"Applies configuration to given hostname using SSH in the format: [<user>[;fingerprint=<host-key fingerprint>]@]<host>[:<port>]",
	)
	targetFlagNames = append(targetFlagNames, "target-ssh")
	cmd.Flags().Uint64Var(
		&sshRekeyThreshold, "target-ssh-rekey-threshold", defaultSshRekeyThreshold,
		"The maximum number of bytes sent or received after which a new key is negotiated. It must be at least 256. If unspecified, a size suitable for the chosen cipher is used.",
	)
	cmd.Flags().StringSliceVar(
		&sshKeyExchanges, "target-ssh-key-exchanges", defaultSshKeyExchanges,
		"The allowed key exchanges algorithms. If unspecified then a default set of algorithms is used. Unsupported values are silently ignored.",
	)
	cmd.Flags().StringSliceVar(
		&sshCiphers, "target-ssh-ciphers", defaultSshCiphers,
		"The allowed cipher algorithms. If unspecified then a sensible default is used. Unsupported values are silently ignored.",
	)
	cmd.Flags().StringSliceVar(
		&sshMACs, "target-ssh-macs", defaultSshMACs,
		"The allowed MAC algorithms. If unspecified then a sensible default is used. Unsupported values are silently ignored.",
	)
	cmd.Flags().StringSliceVar(
		&sshHostKeyAlgorithms, "target-ssh-host-key-algorithms", defaultSshHostKeyAlgorithms,
		"Public key algorithms that the client will accept from the server for host key authentication, in order of preference. If empty, a reasonable default is used.",
	)
	cmd.Flags().DurationVar(
		&sshTcpConnectTimeout, "target-ssh-tcp-connect-timeout", defaultSshTcpConnectTimeout,
		"Timeout is the maximum amount of time for the TCP connection to establish. A Timeout of zero means no timeout.",
	)

	// Docker
	cmd.Flags().StringVarP(
		&docker, "target-docker", "d", defaultDocker,
		"Applies configuration to given Docker container name \n"+
			"Use given format '[<name|uid>[:<group|gid>]@]<image>'",
	)
	targetFlagNames = append(targetFlagNames, "target-docker")

	// Common
	cmd.Flags().BoolVarP(
		&sudo, "target-sudo", "r", defaultSudo,
		"Use sudo to gain root privileges",
	)

	targetFlagNames = append(targetFlagNames, addTargetFlagsArch(cmd)...)

	cmd.MarkFlagsMutuallyExclusive(targetFlagNames...)
	cmd.MarkFlagsOneRequired(targetFlagNames...)
}

func GetHost(ctx context.Context) (types.Host, error) {
	var baseHost types.BaseHost
	var err error

	baseHost, host := getHostArch(ctx)

	if host != nil {
		return host, nil
	} else if baseHost == nil {
		if ssh != "" {
			baseHost, err = hostPkg.NewSshAuthority(ctx, ssh, hostPkg.SshClientConfig{
				RekeyThreshold:    sshRekeyThreshold,
				KeyExchanges:      sshKeyExchanges,
				Ciphers:           sshCiphers,
				MACs:              sshMACs,
				HostKeyAlgorithms: sshHostKeyAlgorithms,
				Timeout:           sshTcpConnectTimeout,
			})
			if err != nil {
				return nil, err
			}
		} else if docker != "" {
			baseHost, err = hostPkg.NewDocker(ctx, docker)
			if err != nil {
				return nil, err
			}
		} else {
			panic("bug: no target set")
		}
	}

	if sudo {
		var err error
		baseHost, err = hostPkg.NewSudoWrapper(ctx, baseHost)
		if err != nil {
			return nil, err
		}
	}

	hst, err := hostPkg.NewAgentClientWrapper(ctx, baseHost)
	if err != nil {
		return nil, err
	}

	return hst, nil
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		ssh = defaultSsh
		sshRekeyThreshold = defaultSshRekeyThreshold
		sshKeyExchanges = defaultSshKeyExchanges
		sshCiphers = defaultSshCiphers
		sshMACs = defaultSshMACs
		sshHostKeyAlgorithms = defaultSshHostKeyAlgorithms
		sshTcpConnectTimeout = defaultSshTcpConnectTimeout
		docker = defaultDocker
		sudo = defaultSudo
	})
}
