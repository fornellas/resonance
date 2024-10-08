package lib

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fornellas/resonance/host"
)

// Implements Host.Run for unix locahost.
func Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	execCmd := exec.CommandContext(ctx, cmd.Path, cmd.Args...)
	if len(cmd.Env) == 0 {
		cmd.Env = []string{"LANG=en_US.UTF-8"}
		for _, value := range os.Environ() {
			if strings.HasPrefix(value, "PATH=") {
				cmd.Env = append(cmd.Env, value)
				break
			}
		}
	}
	execCmd.Env = cmd.Env

	if cmd.Dir == "" {
		cmd.Dir = "/tmp"
	}
	if !filepath.IsAbs(cmd.Dir) {
		return host.WaitStatus{}, &fs.PathError{
			Op:   "Run",
			Path: cmd.Dir,
			Err:  errors.New("path must be absolute"),
		}
	}
	execCmd.Dir = cmd.Dir

	execCmd.Stdin = cmd.Stdin
	execCmd.Stdout = cmd.Stdout
	execCmd.Stderr = cmd.Stderr

	err := execCmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return host.WaitStatus{}, err
		}
	}

	waitStatus := host.WaitStatus{}
	waitStatus.ExitCode = execCmd.ProcessState.ExitCode()
	waitStatus.Exited = execCmd.ProcessState.Exited()
	signal := execCmd.ProcessState.Sys().(syscall.WaitStatus).Signal()
	if signal > 0 {
		waitStatus.Signal = signal.String()
	}
	return waitStatus, nil
}
