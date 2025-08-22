package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"al.essio.dev/pkg/shellescape"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	// docker exec always runs the command over sh, and sh calls env, so, when a command does not
	// exist, either will return 127, and that's the quicky way we can detect os.ErrNotExist here.
	if execCmd.ProcessState.Exited() && execCmd.ProcessState.ExitCode() == 127 {
		return types.WaitStatus{}, os.ErrNotExist
	}

	waitStatus := types.WaitStatus{}
	waitStatus.ExitCode = uint32(execCmd.ProcessState.ExitCode())
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

// GetTestDockerHost creates a Docker BaseHost suitable for usage in tests. It returns the host and
// the docker connection string to it. The container will be purged when the test finishes.
func GetTestDockerHost(t *testing.T, image string) (Docker, string) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker command not found on path")
	}

	ctx, cancel := context.WithCancel(t.Context())
	sanitizedName := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' {
			return r
		}
		return '_'
	}, t.Name())
	name := fmt.Sprintf("resonance-test-%s-%d", sanitizedName, os.Getpid())

	deadline, ok := t.Deadline()
	timeout := 5 * time.Minute
	if !ok {
		timeout = time.Until(deadline)
	}

	cmd := exec.CommandContext(
		ctx,
		"docker", "run",
		"--name", name,
		"--rm",
		image,
		"sleep", fmt.Sprintf("%d", int(timeout.Seconds())),
	)
	stdout := bytes.Buffer{}
	cmd.Stdout = &stdout
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr
	cmd.Start()
	t.Cleanup(func() {
		cancel()
		err := cmd.Wait()
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			if waitStatus, ok := exitError.Sys().(syscall.WaitStatus); ok {
				if waitStatus.Signaled() && waitStatus.Signal() == syscall.SIGKILL {
					return
				}
			}
		}
		assert.NoError(t, err, fmt.Sprintf(
			"docker run returned error:\nstdout:\n%s\nstderr\n%s", stdout.String(), stderr.String()),
		)
	})

	timeoutCh := time.After(timeout)
	for {
		select {
		case <-timeoutCh:
			t.Fatalf("timeout waiting for container")
		case <-time.After(100 * time.Millisecond):
		}
		cmdCheck := exec.CommandContext(ctx, "docker", "exec", name, "/bin/true")
		stdoutBuffer := bytes.Buffer{}
		cmdCheck.Stdout = &stdoutBuffer
		stderrBuffer := bytes.Buffer{}
		cmdCheck.Stderr = &stderrBuffer
		err := cmdCheck.Run()
		if err != nil {
			var exitError *exec.ExitError
			if ok := errors.As(err, &exitError); ok {
				if exitError.Exited() {
					if strings.Contains(stderrBuffer.String(), "No such container") {
						continue
					}
					if strings.Contains(stderrBuffer.String(), "is not running") {
						continue
					}
				}
			}
			require.NoErrorf(
				t, err,
				"failed: %s %s:\nstdout:\n%s\n\nstderr\n%s", cmd.Path, cmd.Args, stdoutBuffer.String(), stderrBuffer.String(),
			)
		}
		break
	}

	connection := fmt.Sprintf("0:0@%s", name)
	dockerHost, err := NewDocker(ctx, connection)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, dockerHost.Close(ctx)) })

	return dockerHost, connection
}
