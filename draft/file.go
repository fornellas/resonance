package draft

import (
	"context"

	"github.com/fornellas/resonance/host/types"
)

// File manages files. Either Absent or exactly one of Socket, SymbolicLink, RegularFile,
// BlockDevice, Directory, CharacterDevice or FIFO must be set. If User and Uid aren't set, then
// Uid = 0 is assumed; either User or Uid can be set (but not both); similar mechanic for Group and
// Gid.
type File struct {
	// Path is the absolute path to the file.
	Path string
	// Whether to remove the file.
	Absent bool
	// Create a socket file.
	Socket bool
	// Create a symbolic link pointing to given path.
	SymbolicLink string
	// Create a regular file with given contents.
	RegularFile *string
	// Create a block device file with given majon / minor.
	BlockDevice *types.FileDevice
	// Create a directory with given contents.
	Directory *[]File
	// Create a character device file with given majon / minor.
	CharacterDevice *types.FileDevice
	// Create a FIFO file.
	FIFO bool
	// Mode bits 07777, see inode(7). Can not be set when SymbolicLink is set.
	Mode *types.FileMode
	// User name owner of the file. If set, then the Uid will attempt to be read from the host
	// during apply.
	User *string
	// User ID owner of the file.
	Uid *uint32
	// Group name owner of the file. If set, then the Gid will attempt to be read from the host
	// during apply.
	Group *string
	// Group ID owner of the file.
	Gid *uint32
}

// Loads the full state of given File path from host.
func LoadFile(ctx context.Context, host types.Host, path string) (*File, error) {
	panic("TODO")
}

func (f *File) ID() string {
	return f.Path
}

func (a *File) Satisfies(ctx context.Context, host types.Host, otherResource Resource) (bool, error) {
	panic("TODO")
}

func (a *File) Validate() error {
	panic("TODO")
}

func (a *File) Merge(otherResource Resource) error {
	panic("TODO")
}

func (a *File) Apply(ctx context.Context, host types.Host) error {
	panic("TODO")
}
