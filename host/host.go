package host

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"strings"
	"time"
)

// Cmd represents a command to be run.
type Cmd struct {
	// Path is the path of the command to run.
	//
	// This is the only field that must be set to a non-zero
	// value. If Path is relative, it is evaluated relative
	// to Dir.
	Path string

	// Args holds command line arguments, including the command as Args[0].
	// If the Args field is empty or nil, Run uses {Path}.
	Args []string

	// Env specifies the environment of the process.
	// Each entry is of the form "key=value".
	// If Env is nil, the new process uses LANG=en_US.UTF-8
	// If Env contains duplicate environment keys, only the last
	// value in the slice for each duplicate key is used.
	Env []string

	// Dir specifies the working directory of the command.
	// If Dir is the empty string, Run runs the command in /tmp
	Dir string

	// Stdin specifies the process's standard input.
	// If Stdin is nil, the remote process reads from an empty
	// bytes.Buffer.
	Stdin io.Reader

	// Stdout and Stderr specify the remote process's standard
	// output and error.
	// If either is nil, Run connects the corresponding file
	// descriptor to an instance of io.Discard.
	// command to block.
	Stdout io.Writer
	Stderr io.Writer
}

func (c Cmd) String() string {
	return fmt.Sprintf("%s %s", c.Path, strings.Join(c.Args, " "))
}

// WaitStatus
type WaitStatus struct {
	// ExitCode returns the exit code of the exited process, or -1 if the process hasn't exited or was terminated by a signal.
	ExitCode int
	// Exited reports whether the program has exited. On Unix systems this reports true if the program exited due to calling exit, but false if the program terminated due to a signal.
	Exited bool
	// Signal describes a process signal.
	Signal string
}

// Success reports whether the program exited successfully, such as with exit status 0 on Unix.
func (ws *WaitStatus) Success() bool {
	return ws.Exited && ws.ExitCode == 0
}

func (ws *WaitStatus) String() string {
	var str string

	if ws.Exited {
		str = fmt.Sprintf("Process exited with status %v", ws.ExitCode)
		if ws.Signal != "" {
			str += fmt.Sprintf(" from signal %v", ws.Signal)
		}
	} else {
		str = "Process did not exit"
		if ws.Signal != "" {
			str += fmt.Sprintf(" from signal %v", ws.Signal)
		}
	}

	return str
}

type HostFileInfo struct {
	Name    string
	Size    int64
	Mode    fs.FileMode
	ModTime time.Time
	IsDir   bool
	Uid     uint32
	Gid     uint32
}

// Host defines an interface for interacting with a host.
type Host interface {
	// Chmod works similar to os.Chmod.
	Chmod(ctx context.Context, name string, mode os.FileMode) error

	// Chown works similar to os.Chown.
	Chown(ctx context.Context, name string, uid, gid int) error

	// // Hostname works similar to os.Hostname.
	// Hostname() (ctx context.Context, name string, err error)

	// // Lchown works similar to os.Lchown.
	// Lchown(ctx context.Context, name string, uid, gid int) error

	// // Link works similar to os.Link.
	// Link(ctx context.Context, oldname, newname string) error

	// Lookup works similar to os/user.Lookup in its pure Go version,
	// that reads from /etc/passwd.
	Lookup(ctx context.Context, username string) (*user.User, error)

	// LookupGroup works similar to os/user.LookupGroup in its pure Go version,
	// that reads from /etc/group.
	LookupGroup(ctx context.Context, name string) (*user.Group, error)

	// Lstat works similar to os.Lstat, but returns HostFileInfo with some
	// extra methods.
	Lstat(ctx context.Context, name string) (HostFileInfo, error)

	// Mkdir works similar to os.Mkdir.
	Mkdir(ctx context.Context, name string, perm os.FileMode) error

	// ReadFile works similar to os.ReadFile.
	ReadFile(ctx context.Context, name string) ([]byte, error)

	// // Readlink works similar to os.Readlink.
	// Readlink(ctx context.Context, name string) (string, error)

	// Remove works similar to os.Remove.
	Remove(ctx context.Context, name string) error

	// Run starts the specified command and waits for it to complete.
	Run(ctx context.Context, cmd Cmd) (WaitStatus, error)

	// // Symlink works similar to os.Symlink.
	// Symlink(ctx context.Context, oldname, newname string) error

	// WriteFile works similar to os.WriteFile.
	WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error

	// A string representation of the host which uniquely identifies it, eg, its FQDN.
	String() string

	// Close any pending connections (if applicable).
	Close() error
}

// Run starts the specified command and waits for it to complete.
// Returns WaitStatus, stdout and stderr.
func Run(ctx context.Context, hst Host, cmd Cmd) (WaitStatus, string, string, error) {
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
