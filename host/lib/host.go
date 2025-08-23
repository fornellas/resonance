package lib

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fornellas/resonance/host/types"
)

// Run starts the specified command and waits for it to complete.
// Returns WaitStatus, stdout and stderr.
func Run(ctx context.Context, baseHost types.BaseHost, cmd types.Cmd) (types.WaitStatus, string, string, error) {
	if cmd.Stdout != nil {
		panic(fmt.Errorf("can not set Cmd.Stdout: %s", cmd))
	}
	stdoutBuffer := bytes.Buffer{}
	cmd.Stdout = &stdoutBuffer

	if cmd.Stderr != nil {
		panic(fmt.Errorf("can not set Cmd.Stderr: %s", cmd))
	}
	stderrBuffer := bytes.Buffer{}
	cmd.Stderr = &stderrBuffer

	waitStatus, err := baseHost.Run(ctx, cmd)
	return waitStatus, stdoutBuffer.String(), stderrBuffer.String(), err
}

// MkdirAll wraps Host.Mkdir and behaves similar to os.MkdirAll.
func MkdirAll(ctx context.Context, host types.Host, name string, mode types.FileMode) error {
	stat_t, err := host.Lstat(ctx, name)
	if err == nil {
		if stat_t.Mode&syscall.S_IFMT == syscall.S_IFDIR {
			return nil
		}
		return &fs.PathError{
			Op:   "MkdirAll",
			Path: name,
			Err:  syscall.ENOTDIR,
		}
	}

	name = filepath.Clean(name)
	parent := filepath.Dir(name)

	if parent != name {
		if err := MkdirAll(ctx, host, parent, mode); err != nil {
			return err
		}
	}

	if err := host.Mkdir(ctx, name, mode); err != nil {
		return err
	}

	return nil
}

// CreateTemp creates a temporary file at the host.Cleanup is responsibility of the caller.
func CreateTemp(ctx context.Context, baseHost types.BaseHost, template string) (string, error) {
	cmd := types.Cmd{
		Path: "mktemp",
		Args: []string{"-t", fmt.Sprintf("%s.XXXXXXXX", template)},
	}
	waitStatus, stdout, stderr, err := Run(ctx, baseHost, cmd)
	if err != nil {
		return "", err
	}
	if !waitStatus.Success() {
		return "", fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	return strings.TrimRight(stdout, "\n"), nil
}
