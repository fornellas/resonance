package types

import "syscall"

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
