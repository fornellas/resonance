package host

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

// Docker uses docker exec to target a running container.
type Docker struct {
	baseRun
	Container string
	User      string
}

func (d Docker) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Run %s", cmd)

	if cmd.Dir == "" {
		cmd.Dir = "/tmp"
	}

	if len(cmd.Env) == 0 {
		cmd.Env = []string{"LANG=en_US.UTF-8"}
		for _, value := range os.Environ() {
			if strings.HasPrefix(value, "PATH=") {
				cmd.Env = append(cmd.Env, value)
				break
			}
		}
	}

	args := []string{"exec"}
	for _, value := range cmd.Env {
		args = append(args, []string{"--env", value}...)
	}
	if cmd.Stdin != nil {
		args = append(args, "--interactive")
	}
	args = append(args, []string{"--user", d.User}...)
	args = append(args, []string{"--workdir", cmd.Dir}...)
	args = append(args, d.Container)
	args = append(args, cmd.Path)
	args = append(args, cmd.Args...)

	execCmd := exec.CommandContext(ctx, "docker", args...)
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

func (d Docker) String() string {
	return fmt.Sprintf("docker:%s", d.Container)
}

func (d Docker) Close() error {
	return nil
}

func NewDocker(ctx context.Context, container, user string) (Docker, error) {
	dockerHst := Docker{
		Container: container,
		User:      user,
	}
	dockerHst.baseRun.Host = &dockerHst
	return dockerHst, nil
}
