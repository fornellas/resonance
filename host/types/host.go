package types

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os/user"
	"path/filepath"
	"syscall"
)

// BaseHost defines a minimalist interface for interfacing with a Linux host.
type BaseHost interface {
	// Run starts the specified command and waits for it to complete.
	Run(ctx context.Context, cmd Cmd) (WaitStatus, error)

	// A string representation of the host which uniquely identifies it, eg, its FQDN.
	String() string

	// String representation for the type of connection is used. eg: ssh, localhost, docker
	Type() string

	// Close any pending connections (if applicable).
	Close(ctx context.Context) error
}

// Host defines a complete interface for interacting with a Linux host
type Host interface {
	BaseHost

	// Geteuid works similar to syscall.Geteuid
	Geteuid(ctx context.Context) (uint64, error)

	// Getegid works similar to syscall.Getegid
	Getegid(ctx context.Context) (uint64, error)

	// Chmod works similar to syscall.Chmod.
	Chmod(ctx context.Context, name string, mode FileMode) error

	// Lchown works similar to syscall.Lchown.
	Lchown(ctx context.Context, name string, uid, gid uint32) error

	// Hostname works similar to os.Hostname.
	// Hostname() (ctx context.Context, name string, err error)

	// Lookup works similar to os/user.Lookup in its pure Go version,
	// that reads from /etc/passwd.
	Lookup(ctx context.Context, username string) (*user.User, error)

	// LookupGroup works similar to os/user.LookupGroup in its pure Go version,
	// that reads from /etc/group.
	LookupGroup(ctx context.Context, name string) (*user.Group, error)

	// Lstat works similar to syscal.Lstat
	Lstat(ctx context.Context, name string) (*Stat_t, error)

	// ReadDir reads the named directory, returning all its DirEnt.
	ReadDir(ctx context.Context, name string) (dirEntResultCh <-chan DirEntResult, cancel func())

	// Mkdir works similar to syscall.Mkdir, but ignoring umask.
	Mkdir(ctx context.Context, name string, mode FileMode) error

	// ReadFile works similar to os.ReadFile.
	ReadFile(ctx context.Context, name string) (io.ReadCloser, error)

	// Symlink works similar to syscall.Symlink.
	Symlink(ctx context.Context, oldname, newname string) error

	// Readlink works similar to os.Readlink.
	Readlink(ctx context.Context, name string) (string, error)

	// Remove works similar to os.Remove.
	Remove(ctx context.Context, name string) error

	// Mknod works similar to syscall.Mknod, but ignoring umask.
	Mknod(ctx context.Context, path string, mode FileMode, dev FileDevice) error

	// WriteFile works similar to os.WriteFile, but receives mode bits (see inode(7)) and ignores umask.
	WriteFile(ctx context.Context, name string, data io.Reader, mode FileMode) error
}

// Run starts the specified command and waits for it to complete.
// Returns WaitStatus, stdout and stderr.
func Run(ctx context.Context, hst BaseHost, cmd Cmd) (WaitStatus, string, string, error) {
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
func MkdirAll(ctx context.Context, hst Host, name string, mode FileMode) error {
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
