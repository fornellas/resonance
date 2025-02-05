package host

import (
	"fmt"
	"io"
	"strings"
	"syscall"
)

var DefaultPath = "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

var DefaultEnv = []string{
	"LANG=en_US.UTF-8",
	DefaultPath,
}

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
	// If Env is nil, the new process uses DefaultEnv.
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
		str = "Process did not exit:"
		if ws.Signal != "" {
			str += fmt.Sprintf(" received signal: %#v", ws.Signal)
		}
	}

	return str
}

// Timespec from syscall.Timespec for Linux
type Timespec struct {
	Sec  int64
	Nsec int64
}

// Stat_t from syscall.Stat_t for Linux
type Stat_t struct {
	Dev     uint64
	Ino     uint64
	Nlink   uint64
	Mode    uint32
	Uid     uint32
	Gid     uint32
	Rdev    uint64
	Size    int64
	Blksize int64
	Blocks  int64
	Atim    Timespec
	Mtim    Timespec
	Ctim    Timespec
}

// Dirent is similar to syscall.Dirent
type DirEnt struct {
	Ino  uint64
	Type uint8
	Name string
}

func (d *DirEnt) IsBlockDevice() bool {
	return d.Type == syscall.DT_BLK
}

func (d *DirEnt) IsCharacterDevice() bool {
	return d.Type == syscall.DT_CHR
}

func (d *DirEnt) IsDirectory() bool {
	return d.Type == syscall.DT_DIR
}

func (d *DirEnt) IsFIFO() bool {
	return d.Type == syscall.DT_FIFO
}

func (d *DirEnt) IsSymbolicLink() bool {
	return d.Type == syscall.DT_LNK
}

func (d *DirEnt) IsRegularFile() bool {
	return d.Type == syscall.DT_REG
}

func (d *DirEnt) IsUnixDomainSocket() bool {
	return d.Type == syscall.DT_SOCK
}

type DirEntResult struct {
	DirEnt DirEnt
	Error  error
}
