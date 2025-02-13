package main

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host/types"
	storePkg "github.com/fornellas/resonance/store"
)

// This is to be used in place of os.Exit() to aid writing test assertions on exit code.
var Exit func(int) = func(code int) { os.Exit(code) }

var ssh string
var defaultSsh = ""

var docker string
var defaultDocker = ""

var sudo bool
var defaultSudo = false

var storeValue = NewStoreValue()

var storeHostTargetPath string
var defaultStoreHostTargetPath = "/var/lib/resonance"

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

func addCommonTargetFlags(cmd *cobra.Command) []string {
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

	return targetFlagNames
}

func addStoreFlagsCommon(cmd *cobra.Command) {
	cmd.PersistentFlags().VarP(storeValue, "store", "", "Where to store state information")

	cmd.Flags().StringVarP(
		&storeHostTargetPath, "store-target-path", "", defaultStoreHostTargetPath,
		"Path on target host where to store state",
	)
}

func getStoreCommon(hst types.Host) storePkg.Store {
	var storePath string
	switch storeValue.String() {
	case "target":
		storePath = storeHostTargetPath
	default:
		return nil
	}
	return storeValue.GetStore(hst, storePath)
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		ssh = defaultSsh
		docker = defaultDocker
		storeHostTargetPath = defaultStoreHostTargetPath
		sudo = defaultSudo
	})
}
