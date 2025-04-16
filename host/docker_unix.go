package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"al.essio.dev/pkg/shellescape"

	"github.com/fornellas/resonance/host/types"
)

// Docker uses docker exec to target a running container.
type Docker struct {
	// User/group and image in the format "[<name|uid>[:<group|gid>]@]<image>" (eg: root@ubuntu)
	ConnectionString string
}

func NewDocker(ctx context.Context, connection string) (Docker, error) {
	dockerHst := Docker{
		ConnectionString: connection,
	}
	return dockerHst, nil
}

func (h Docker) Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
	parts := strings.Split(h.ConnectionString, "@")
	var dockerConnectionUser, dockerConnectionContainer string
	switch len(parts) {
	case 1:
		dockerConnectionUser = "0:0"
		dockerConnectionContainer = parts[0]
	case 2:
		dockerConnectionUser = parts[0]
		dockerConnectionContainer = parts[1]
	default:
		return types.WaitStatus{}, fmt.Errorf("invalid connection string format: %s", h.ConnectionString)
	}

	if cmd.Dir == "" {
		cmd.Dir = "/tmp"
	}
	if !filepath.IsAbs(cmd.Dir) {
		return types.WaitStatus{}, &fs.PathError{
			Op:   "Run",
			Path: cmd.Dir,
			Err:  errors.New("path must be absolute"),
		}
	}

	if len(cmd.Env) == 0 {
		cmd.Env = types.DefaultEnv
	}

	args := []string{"exec"}
	if cmd.Stdin != nil {
		args = append(args, "--interactive")
	}
	args = append(args, []string{"--user", dockerConnectionUser}...)
	args = append(args, []string{"--workdir", cmd.Dir}...)
	args = append(args, dockerConnectionContainer)

	cmdStr := []string{
		"env", "-i",
	}
	for _, env := range cmd.Env {
		cmdStr = append(cmdStr, shellescape.Quote(env))
	}
	cmdStr = append(cmdStr, shellescape.Quote(cmd.Path))
	for _, arg := range cmd.Args {
		cmdStr = append(cmdStr, shellescape.Quote(arg))
	}
	args = append(args, "sh", "-c", strings.Join(cmdStr, " "))

	execCmd := exec.CommandContext(ctx, "docker", args...)
	execCmd.Stdin = cmd.Stdin
	execCmd.Stdout = cmd.Stdout
	execCmd.Stderr = cmd.Stderr

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

func (h Docker) String() string {
	return h.ConnectionString
}

func (h Docker) Type() string {
	return "docker"
}

func (h Docker) Close(ctx context.Context) error {
	return nil
}
