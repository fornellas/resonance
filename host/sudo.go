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
	cmd.Args = append([]string{cmd.Path}, cmd.Args...)
	cmd.Path = "sudo"
	return s.Host.Run(ctx, cmd)
}

func (s Sudo) String() string {
	return fmt.Sprintf("%s(sudo)", s.Host.String())
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
		Path:  "true",
		Stdin: os.Stdin,
	}
	waitStatus, stdout, stderr, err := sudoHost.Run(nestedCtx, cmd)
	if err != nil {
		return Sudo{}, err
	}
	if !waitStatus.Success() {
		return Sudo{}, fmt.Errorf(
			"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus, stdout, stderr,
		)
	}

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
			"failed to run %s: %v\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus, stdout, stderr,
		)
	}

	return sudoHost, nil
}
