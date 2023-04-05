package host

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fornellas/resonance/log"
)

// Local interacts with the local machine running the code.
type Local struct{}

func (l Local) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Chmod %v %s", mode, name)
	return os.Chmod(name, mode)
}

func (l Local) Chown(ctx context.Context, name string, uid, gid int) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Chown %v %v %s", uid, gid, name)
	return os.Chown(name, uid, gid)
}

func (l Local) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Lookup %s", username)
	return user.Lookup(username)
}

func (l Local) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("LookupGroup %s", name)
	return user.LookupGroup(name)
}

func (l Local) Lstat(ctx context.Context, name string) (HostFileInfo, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Lstat %s", name)
	fileInfo, err := os.Lstat(name)
	if err != nil {
		return HostFileInfo{}, err
	}
	stat_t := fileInfo.Sys().(*syscall.Stat_t)
	return NewHostFileInfo(
		filepath.Base(name),
		fileInfo.Size(),
		fileInfo.Mode(),
		fileInfo.ModTime(),
		fileInfo.IsDir(),
		stat_t.Uid,
		stat_t.Gid,
	), nil
}

func (l Local) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Mkdir %s", name)
	return os.Mkdir(name, perm)
}

func (l Local) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("ReadFile %s", name)
	return os.ReadFile(name)
}

func (l Local) Remove(ctx context.Context, name string) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Remove %s", name)
	return os.Remove(name)
}

func (l Local) Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Run %s", cmd)

	execCmd := exec.CommandContext(log.IndentLogger(ctx), cmd.Path, cmd.Args...)
	if len(cmd.Env) == 0 {
		cmd.Env = []string{"LANG=en_US.UTF-8"}
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

	waitStatus := WaitStatus{}
	err := execCmd.Run()

	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return waitStatus, err
		}
	}
	waitStatus.ExitCode = execCmd.ProcessState.ExitCode()
	waitStatus.Exited = execCmd.ProcessState.Exited()
	signal := execCmd.ProcessState.Sys().(syscall.WaitStatus).Signal()
	if signal > 0 {
		waitStatus.Signal = signal.String()
	}
	return waitStatus, nil
}

func (l Local) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("WriteFile %s %v", name, perm)
	return os.WriteFile(name, data, perm)
}

func (l Local) String() string {
	return "localhost"
}

func (l Local) Close() error {
	return nil
}
