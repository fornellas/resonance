package lib

// This lib package contains common functions used at various places.
// These can't live at the host package, because if the agent imports the host package, its size
// baloons, so we have just a handful of functions at lib.

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"syscall"

	"github.com/fornellas/resonance/host/types"
)

// SimpleRun starts the specified command and waits for it to complete.
// Returns WaitStatus, stdout and stderr.
func SimpleRun(ctx context.Context, hst types.BaseHost, cmd types.Cmd) (types.WaitStatus, string, string, error) {
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

	waitStatus, err := hst.Run(ctx, cmd)
	return waitStatus, stdoutBuffer.String(), stderrBuffer.String(), err
}

// MkdirAll wraps Host.Mkdir and behavess similar to os.MkdirAll.
func MkdirAll(ctx context.Context, hst types.Host, name string, mode uint32) error {
	stat_t, err := hst.Lstat(ctx, name)
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
		if err := MkdirAll(ctx, hst, parent, mode); err != nil {
			return err
		}
	}

	if err := hst.Mkdir(ctx, name, mode); err != nil {
		return err
	}

	return nil
}
