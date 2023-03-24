package host

import (
	"context"
	"fmt"
	"os"

	"github.com/fornellas/resonance/log"
)

// Sudo implements Host interface by having all methods rely on an underlying Host.Run, and
// preceding all commands with sudo.
type Sudo struct {
	baseRun
	Host Host
}

func (s Sudo) Run(ctx context.Context, cmd Cmd) (WaitStatus, string, string, error) {
	cmd.Args = append([]string{"--non-interactive", "--", cmd.Path}, cmd.Args...)
	cmd.Path = "sudo"
	return s.Host.Run(ctx, cmd)
}

func (s Sudo) String() string {
	return s.Host.String()
}

func (s Sudo) Close() error {
	return s.Host.Close()
}

func NewSudo(ctx context.Context, host Host) (Sudo, error) {
	logger := log.GetLogger(ctx)
	logger.Info("⚡ Sudo access")
	nestedCtx := log.IndentLogger(ctx)

	sudoHost := Sudo{
		Host: host,
	}
	sudoHost.baseRun.Host = sudoHost

	// Sudo MAY ask for password once
	cmd := Cmd{
		Path:  "sudo",
		Args:  []string{"--", "true"},
		Stdin: os.Stdin,
	}
	waitStatus, stdout, stderr, err := sudoHost.Host.Run(nestedCtx, cmd)
	if err != nil {
		return Sudo{}, err
	}
	if !waitStatus.Success() {
		return Sudo{}, fmt.Errorf(
			"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus, stdout, stderr,
		)
	}

	logger.Debug("pass!")

	// Sudo must NOT ask for password again
	cmd = Cmd{
		Path: "true",
	}
	waitStatus, stdout, stderr, err = sudoHost.Run(nestedCtx, cmd)
	if err != nil {
		return Sudo{}, err
	}
	if !waitStatus.Success() {
		return Sudo{}, fmt.Errorf(
			"sudo is still asking for a password: failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}

	return sudoHost, nil
}
