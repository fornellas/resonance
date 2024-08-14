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
	cmdHost
<<<<<<< HEAD
	// User/group and image in the format "[<name|uid>[:<group|gid>]@]<image>" (eg: root@ubuntu)
	ConnectionString string
=======
	// Username or UID (format: "<name|uid>[:<group|gid>]") and Container Name ( eg: root@ubuntu )
	Connection string
>>>>>>> eee95cc (chore: Define docker string as a single parameter for the Command Line)
}

func NewDocker(ctx context.Context, connection string) (Docker, error) {
	dockerHst := Docker{
<<<<<<< HEAD
		ConnectionString: connection,
=======
		Connection: connection,
>>>>>>> eee95cc (chore: Define docker string as a single parameter for the Command Line)
	}
	dockerHst.cmdHost.Host = &dockerHst
	return dockerHst, nil
}

func (d Docker) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Run", "cmd", cmd)

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

<<<<<<< HEAD
	parts := strings.Split(d.ConnectionString, "@")

	var dockerConnectionUser, dockerConnectionContainer string
	switch len(parts) {
	case 1:
		dockerConnectionUser = "0:0"
		dockerConnectionContainer = parts[0]
	case 2:
		dockerConnectionUser = parts[0]
		dockerConnectionContainer = parts[1]
	default:
		return host.WaitStatus{}, fmt.Errorf("invalid connection string format: %s", d.ConnectionString)
	}
=======
	dockerConnectionUser := strings.Split(d.Connection, "@")[0]
	dockerConnectionContainer := strings.Split(d.Connection, "@")[1]
>>>>>>> eee95cc (chore: Define docker string as a single parameter for the Command Line)

	args = append(args, []string{"--user", dockerConnectionUser}...)
	args = append(args, []string{"--workdir", cmd.Dir}...)
	args = append(args, dockerConnectionContainer)
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
<<<<<<< HEAD
	return fmt.Sprintf(d.ConnectionString)
}

func (d Docker) Type() string {
	return d.Host.Type()
=======
	return fmt.Sprintf("%s", d.Connection)
}

func (d Docker) Type() string {
	return "docker"
>>>>>>> eee95cc (chore: Define docker string as a single parameter for the Command Line)
}

func (d Docker) Close() error {
	return nil
}
