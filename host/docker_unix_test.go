package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func GetTestDockerHost(t *testing.T, image string) (Docker, string) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker command not found on path")
	}

	ctx, cancel := context.WithCancel(t.Context())

	name := fmt.Sprintf("resonance-test-%s-%d", t.Name(), os.Getpid())
	cmd := exec.CommandContext(
		ctx,
		"docker", "run",
		"--name", name,
		"--rm",
		image,
		"sleep", "5",
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

	timeoutCh := time.After(2 * time.Minute)
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

func TestDocker(t *testing.T) {
	dockerHost, connection := GetTestDockerHost(t, "debian")

	testBaseHost(t, dockerHost, connection, "docker")
}
