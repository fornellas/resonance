package local

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/fornellas/resonance/host/types"
)

func Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
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
	execCmd.Dir = cmd.Dir

	execCmd.Stdin = cmd.Stdin
	execCmd.Stdout = cmd.Stdout
	execCmd.Stderr = cmd.Stderr

	execCmd.Cancel = func() error {
		if err := execCmd.Process.Signal(syscall.SIGTERM); err != nil {
			return err
		}
		time.Sleep(3 * time.Second)
		// process may have exited by now, should be safe-ish to ignore errors here
		execCmd.Process.Signal(syscall.SIGKILL)
		return nil
	}

	err := execCmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return types.WaitStatus{}, err
		}
	}

	waitStatus := types.WaitStatus{}
	waitStatus.ExitCode = execCmd.ProcessState.ExitCode()
	waitStatus.Exited = execCmd.ProcessState.Exited()
	signal := execCmd.ProcessState.Sys().(syscall.WaitStatus).Signal()
	if signal > 0 {
		waitStatus.Signal = signal.String()
	}
	return waitStatus, nil
}
