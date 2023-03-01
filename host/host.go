package host

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
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
	//
	// If Stdin is nil, the process reads from the null device (os.DevNull).
	//
	// If Stdin is an *os.File, the process's standard input is connected
	// directly to that file.
	//
	// Otherwise, during the execution of the command a separate
	// goroutine reads from Stdin and delivers that data to the command
	// over a pipe. In this case, Wait does not complete until the goroutine
	// stops copying, either because it has reached the end of Stdin
	// (EOF or a read error), or because writing to the pipe returned an error,
	// or because a nonzero WaitDelay was set and expired.
	Stdin io.Reader
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
		str := fmt.Sprintf("Process exited with status %v", ws.ExitCode)
		if ws.Signal != "" {
			str += fmt.Sprintf(" from signal %v", ws.Signal)
		}
	} else {
		str := "Process did not exit"
		if ws.Signal != "" {
			str += fmt.Sprintf(" with signal %v", ws.Signal)
		}
	}

	return str
}

// Host defines an interface for interacting with a host.
type Host interface {
	// // Chmod works similar to os.Chmod.
	// Chmod(ctx context.Context, name string, mode os.FileMode) error

	// // Chown works similar to os.Chown.
	// Chown(ctx context.Context, name string, uid, gid int) error

	// // GetGid works similar to os.GetGid.
	// GetGid(ctx context.Context, groupname string) int

	// // GetUid works similar to os.GetUid.
	// GetUid(ctx context.Context, username string) int

	// // Hostname works similar to os.Hostname.
	// Hostname() (ctx context.Context, name string, err error)

	// // Lchown works similar to os.Lchown.
	// Lchown(ctx context.Context, name string, uid, gid int) error

	// // Link works similar to os.Link.
	// Link(ctx context.Context, oldname, newname string) error

	// Lstat works similar to os.Lstat, but it always returns non-nil Sys().
	Lstat(ctx context.Context, name string) (os.FileInfo, error)

	// // Mkdir works similar to os.Mkdir.
	// Mkdir(ctx context.Context, name string, perm os.FileMode) error

	// ReadFile works similar to os.ReadFile.
	ReadFile(ctx context.Context, name string) ([]byte, error)

	// // Readlink works similar to os.Readlink.
	// Readlink(ctx context.Context, name string) (string, error)

	// Remove works similar to os.Remove.
	Remove(ctx context.Context, name string) error

	// Run starts the specified command and waits for it to complete.
	// Returns WaitStatus, stdout, stderr, error
	Run(ctx context.Context, cmd Cmd) (WaitStatus, string, string, error)

	// // Symlink works similar to os.Symlink.
	// Symlink(ctx context.Context, oldname, newname string) error

	// // WriteFile works similar to os.WriteFile.
	// WriteFile(ctx context.Context, path string, perm os.FileMode)

	String() string
}
