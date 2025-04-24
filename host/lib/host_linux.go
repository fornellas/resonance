package lib

// This lib package contains common functions used at various places.
// These can't live at the host package, because if the agent imports the host package, its size
// baloons, so we have just a handful of functions at lib.

import (
	"context"
	"errors"
	"io/fs"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/fornellas/resonance/host/types"
)

// Implements Host.Run for unix locahost.
func LocalRun(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
	execCmd := exec.CommandContext(ctx, cmd.Path, cmd.Args...)
	if len(cmd.Env) == 0 {
		execCmd.Env = types.DefaultEnv
	} else {
		execCmd.Env = cmd.Env
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
	execCmd.Dir = cmd.Dir

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

// Implements Host.ReadDir for Linux locahost.
func LocalReadDir(ctx context.Context, name string) (<-chan types.DirEntResult, func()) {
	ctx, cancel := context.WithCancel(ctx)

	dirEntResultCh := make(chan types.DirEntResult, 100)

	go func() {
		if !filepath.IsAbs(name) {
			dirEntResultCh <- types.DirEntResult{
				Error: &fs.PathError{
					Op:   "ReadDir",
					Path: name,
					Err:  errors.New("path must be absolute"),
				},
			}
			close(dirEntResultCh)
			return
		}

		fd, err := syscall.Open(name, syscall.O_RDONLY, 0)
		if err != nil {
			dirEntResultCh <- types.DirEntResult{
				Error: &fs.PathError{
					Op:   "ReadDir",
					Path: name,
					Err:  err,
				},
			}
			close(dirEntResultCh)
			return
		}
		defer syscall.Close(fd)

		buf := make([]byte, 8196)

		for {
			// We do this via syscall.Getdents instead of os.ReadDir, because the latter
			// requires doing aditional stat calls, which is slower.
			n, err := syscall.Getdents(fd, buf)
			if err != nil {
				dirEntResultCh <- types.DirEntResult{
					Error: &fs.PathError{
						Op:   "ReadDir",
						Path: name,
						Err:  err,
					},
				}
				break
			}

			if n == 0 {
				break
			}

			offset := 0
			for offset < n {
				dirent := (*syscall.Dirent)(unsafe.Pointer(&buf[offset]))

				var l int
				for l := range len(dirent.Name) {
					if dirent.Name[l] == 0 {
						break
					}
				}
				nameBytes := make([]byte, l)
				for i := range l {
					nameBytes[i] = byte(dirent.Name[i])
				}
				name := string(nameBytes)

				if name != "." && name != ".." {
					dirEnt := types.DirEnt{
						Ino:  dirent.Ino,
						Type: dirent.Type,
						Name: name,
					}

					select {
					case dirEntResultCh <- types.DirEntResult{DirEnt: dirEnt}:
					case <-ctx.Done():
						close(dirEntResultCh)
						return
					}
				}

				offset += int(dirent.Reclen)
			}
		}

		close(dirEntResultCh)
	}()

	return dirEntResultCh, cancel
}
