package types

import (
	"fmt"
	"log/slog"
	"syscall"

	"golang.org/x/sys/unix"
)

// File mode bits 07777, see inode(7).
type FileMode uint32

func (f *FileMode) String() string {
	if *f > 0 {
		return fmt.Sprintf("0%o", *f)
	}
	return "0"
}

func (f FileMode) LogValue() slog.Value {
	return slog.StringValue(f.String())
}

// Mask for all file mode bits.
var FileModeBitsMask FileMode = 07777

// File device dev_t, as expected by mknod(2).
type FileDevice uint64

func (f *FileDevice) String() string {
	return fmt.Sprintf("%d,%d", unix.Major(uint64(*f)), unix.Minor(uint64(*f)))
}

func (f FileDevice) LogValue() slog.Value {
	return slog.StringValue(f.String())
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
