package host

import (
	"bytes"
	"context"
	"fmt"
	"os/user"
)

// Host defines an interface for interacting with a host
type Host interface {
	// Chmod works similar to syscall.Chmod.
	Chmod(ctx context.Context, name string, mode uint32) error

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

	// Lstat works similar to syscal.Lstat
	Lstat(ctx context.Context, name string) (*Stat_t, error)

	// Mkdir works similar to syscall.Mkdir, but no umask is applied.
	Mkdir(ctx context.Context, name string, mode uint32) error

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

	// WriteFile works similar to os.WriteFile, but receives mode as is syscall.Chmod argument.
	WriteFile(ctx context.Context, name string, data []byte, mode uint32) error

	// A string representation of the host which uniquely identifies it, eg, its FQDN.
	String() string

	// String representation for the type of connection is used. eg: ssh, localhost, docker
	Type() string

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
