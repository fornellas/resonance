package resources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"syscall"

	"github.com/fornellas/resonance/host/types"
)

// File manages files. One of Absent, Socket, SymbolicLink, RegularFile, BlockDevice, Directory,
// CharacterDevice or FIFO must be set. If User and Uid aren't set, then Uid = 0 is assumed; either
// User or Uid can be set (but not both); similar mechanic for Group and Gid.
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
	Directory *[]*File
	// Create a character device file with given majon / minor.
	CharacterDevice *types.FileDevice
	// Create a FIFO file.
	FIFO bool
	// Mode bits 07777, see inode(7). Can not be set when SymbolicLink is set.
	Mode *types.FileMode
	// User name owner of the file. If set, then the Uid will attempt to be read from the host.
	User *string
	// User ID owner of the file.
	Uid *uint32
	// Group name owner of the file. If set, then the Gid will attempt to be read from the host.
	Group *string
	// Group ID owner of the file.
	Gid *uint32
}

func loadFileSymbolicLink(ctx context.Context, host types.Host, file *File) error {
	target, err := host.Readlink(ctx, file.Path)
	if err != nil {
		return err
	}
	file.SymbolicLink = target
	file.Mode = nil
	return nil
}

func loadFileRegularFile(ctx context.Context, host types.Host, file *File) error {
	fileReadCloser, err := host.ReadFile(ctx, string(file.Path))
	if err != nil {
		return err
	}
	contentBytes, err := io.ReadAll(fileReadCloser)
	if err != nil {
		if closeErr := fileReadCloser.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return err
	}
	if err := fileReadCloser.Close(); err != nil {
		return err
	}
	file.RegularFile = new(string)
	*file.RegularFile = string(contentBytes)
	return nil
}

func loadFileDirectory(ctx context.Context, host types.Host, file *File) error {
	dirEntResultCh, cancel := host.ReadDir(ctx, file.Path)
	defer cancel()

	directory := []*File{}
	file.Directory = &directory
	for dirEntResult := range dirEntResultCh {
		if dirEntResult.Error != nil {
			return dirEntResult.Error
		}
		path := filepath.Join(file.Path, dirEntResult.DirEnt.Name)
		subFile, err := LoadFile(ctx, host, path)
		if err != nil {
			return err
		}
		directory = append(directory, subFile)
	}

	sort.SliceStable(directory, func(i, j int) bool {
		return directory[i].Path < directory[j].Path
	})

	return nil
}

// Loads the full state of given File path from host.
func LoadFile(ctx context.Context, host types.Host, path string) (*File, error) {
	file := &File{
		Path: path,
	}

	stat_t, err := host.Lstat(ctx, string(file.Path))
	if err != nil {
		if os.IsNotExist(err) {
			file.Absent = true
			return file, nil
		}
		return nil, err
	}

	var mode types.FileMode = types.FileMode(stat_t.Mode) & types.FileModeBitsMask
	file.Mode = &mode
	file.Uid = &stat_t.Uid
	file.Gid = &stat_t.Gid

	switch stat_t.Mode & syscall.S_IFMT {
	case syscall.S_IFSOCK:
		file.Socket = true
	case syscall.S_IFLNK:
		if err := loadFileSymbolicLink(ctx, host, file); err != nil {
			return nil, err
		}
	case syscall.S_IFREG:
		if err := loadFileRegularFile(ctx, host, file); err != nil {
			return nil, err
		}
	case syscall.S_IFBLK:
		file.BlockDevice = (*types.FileDevice)(&stat_t.Rdev)
	case syscall.S_IFDIR:
		if err := loadFileDirectory(ctx, host, file); err != nil {
			return nil, err
		}
	case syscall.S_IFCHR:
		file.CharacterDevice = (*types.FileDevice)(&stat_t.Rdev)
	case syscall.S_IFIFO:
		file.FIFO = true
	default:
		panic(fmt.Sprintf("bug: unexpected stat_t.Mode: 0x%x", stat_t.Mode))
	}

	return file, nil
}

func (f *File) ID() string {
	return f.Path
}

func (f *File) getUid(ctx context.Context, host types.Host) (uint32, error) {
	if f.Uid != nil {
		return *f.Uid, nil
	}

	if f.User == nil {
		return 0, nil
	}

	user, err := host.Lookup(ctx, *f.User)
	if err != nil {
		return 0, err
	}

	uidUint64, err := strconv.ParseUint(user.Uid, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse UID: %s: %w", user.Uid, err)
	}
	return uint32(uidUint64), nil
}

func (f *File) getGid(ctx context.Context, host types.Host) (uint32, error) {
	if f.Gid != nil {
		return *f.Gid, nil
	}

	if f.Group == nil {
		return 0, nil
	}

	group, err := host.LookupGroup(ctx, *f.Group)
	if err != nil {
		return 0, err
	}

	gidUint64, err := strconv.ParseUint(group.Gid, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse GID: %s: %w", group.Gid, err)
	}
	return uint32(gidUint64), nil
}

func (f *File) satisfiesDirectory(ctx context.Context, host types.Host, otherFile *File) (bool, error) {
	if otherFile.Directory != nil {
		if f.Directory == nil {
			return false, nil
		}
		for _, otherDirFile := range *otherFile.Directory {
			found := false
			for _, dirFile := range *f.Directory {
				if dirFile.ID() == otherDirFile.ID() {
					satisfied, err := dirFile.Satisfies(ctx, host, otherDirFile)
					if err != nil {
						return false, err
					}
					if !satisfied {
						return false, nil
					}
					found = true
					break
				}
			}
			if !found {
				return false, nil
			}
		}
	}
	return true, nil
}

func (f *File) satisfiesTypes(ctx context.Context, host types.Host, otherFile *File) (bool, error) {
	typeCheckFns := []func() (bool, error){
		// Socket
		func() (bool, error) { return otherFile.Socket && !f.Socket, nil },
		// SymbolicLink
		func() (bool, error) {
			return len(otherFile.SymbolicLink) > 0 && otherFile.SymbolicLink != f.SymbolicLink, nil
		},
		// RegularFile
		func() (bool, error) {
			return otherFile.RegularFile != nil && (f.RegularFile == nil || (*otherFile.RegularFile != *f.RegularFile)), nil
		},
		// BlockDevice
		func() (bool, error) {
			return otherFile.BlockDevice != nil && (f.BlockDevice == nil || (*otherFile.BlockDevice != *f.BlockDevice)), nil
		},
		// Directory
		func() (bool, error) { return f.satisfiesDirectory(ctx, host, otherFile) },
		// CharacterDevice
		func() (bool, error) {
			return otherFile.CharacterDevice != nil && (f.CharacterDevice == nil || (*otherFile.CharacterDevice != *f.CharacterDevice)), nil
		},
		// FIFO
		func() (bool, error) { return otherFile.FIFO && !f.FIFO, nil },
	}

	for _, typeCheckFn := range typeCheckFns {
		satisfies, err := typeCheckFn()
		if err != nil {
			return false, err
		}
		if !satisfies {
			return false, nil
		}
	}

	return true, nil
}

func (f *File) Satisfies(ctx context.Context, host types.Host, otherResource Resource) (bool, error) {
	otherFile := otherResource.(*File)
	// Path
	if otherFile.Path != f.Path {
		return false, nil
	}
	// Absent
	if otherFile.Absent && !f.Absent {
		return false, nil
	}
	// Types
	satisfies, err := f.satisfiesTypes(ctx, host, otherFile)
	if err != nil {
		return false, err
	}
	if !satisfies {
		return false, nil
	}
	// Mode
	if otherFile.Mode != nil && (f.Mode == nil || (*otherFile.Mode != *f.Mode)) {
		return false, nil
	}
	// User / Uid
	otherUid, err := otherFile.getUid(ctx, host)
	if err != nil {
		return false, err
	}
	uid, err := f.getUid(ctx, host)
	if err != nil {
		return false, err
	}
	if otherUid != uid {
		return false, nil
	}
	// Group / Gid
	otherGid, err := otherFile.getGid(ctx, host)
	if err != nil {
		return false, err
	}
	gid, err := f.getGid(ctx, host)
	if err != nil {
		return false, err
	}
	if otherGid != gid {
		return false, nil
	}

	return true, nil
}

func (f *File) validatePath() error {
	if !isCleanUnixPath(f.Path) {
		return fmt.Errorf("'path' must be a clean unix path")
	}
	if !isAbsUnixPath(f.Path) {
		return fmt.Errorf("'path' must be an absolute unix path")
	}
	return nil
}

func (f *File) validateAbsent() (bool, error) {
	if f.Absent {
		if f.Socket {
			return false, fmt.Errorf("'socket' can not be set with 'absent'")
		}
		if len(f.SymbolicLink) > 0 {
			return false, fmt.Errorf("'symbolic_link' can not be set with 'absent'")
		}
		if f.RegularFile != nil {
			return false, fmt.Errorf("'regular_file' can not be set with 'absent'")
		}
		if f.BlockDevice != nil {
			return false, fmt.Errorf("'block_device' can not be set with 'absent'")
		}
		if f.Directory != nil {
			return false, fmt.Errorf("'directory' can not be set with 'absent'")
		}
		if f.CharacterDevice != nil {
			return false, fmt.Errorf("'character_device' can not be set with 'absent'")
		}
		if f.FIFO {
			return false, fmt.Errorf("'fifo' can not be set with 'absent'")
		}
		if f.Mode != nil {
			return false, fmt.Errorf("'mode' can not be set with 'absent'")
		}
		if f.User != nil {
			return false, fmt.Errorf("'user' can not be set with 'absent'")
		}
		if f.Uid != nil {
			return false, fmt.Errorf("'uid' can not be set with 'absent'")
		}
		if f.Group != nil {
			return false, fmt.Errorf("'group' can not be set with 'absent'")
		}
		if f.Gid != nil {
			return false, fmt.Errorf("'gid' can not be set with 'absent'")
		}
		return true, nil
	}
	return false, nil
}

func (f *File) validateDirectory() error {
	for _, subFile := range *f.Directory {
		if dirUnix(subFile.Path) != f.Path {
			return fmt.Errorf("directory entry '%s' is not a subpath of '%s'", subFile.Path, f.Path)
		}
		if err := subFile.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (f *File) validateType() (string, error) {
	fileTypeCount := 0
	failModeWith := ""

	// Socket
	if f.Socket {
		fileTypeCount += 1
	}

	// SymbolicLink
	if len(f.SymbolicLink) > 0 {
		fileTypeCount += 1
		failModeWith = "symbolic_link"
		if !isCleanUnixPath(f.SymbolicLink) {
			return "", fmt.Errorf("'symbolic_link' must be a clean unix path")
		}
	}

	// RegularFile
	if f.RegularFile != nil {
		fileTypeCount += 1
	}

	// BlockDevice
	if f.BlockDevice != nil {
		fileTypeCount += 1
	}

	// Directory
	if f.Directory != nil {
		fileTypeCount += 1
		if err := f.validateDirectory(); err != nil {
			return "", err
		}
	}

	// CharacterDevice
	if f.CharacterDevice != nil {
		fileTypeCount += 1
	}

	// FIFO
	if f.FIFO {
		fileTypeCount += 1
	}

	if fileTypeCount != 1 {
		return "", fmt.Errorf("exactly one file type can be set: 'socket', 'symbolic_link', 'regular_file', 'block_device', 'directory', 'character_device' or 'fifo'")
	}

	return failModeWith, nil
}

func (f *File) Validate() error {
	// Path
	if err := f.validatePath(); err != nil {
		return err
	}

	// Absent
	finished, err := f.validateAbsent()
	if err != nil {
		return err
	}
	if finished {
		return nil
	}

	// Types
	failModeWith, err := f.validateType()
	if err != nil {
		return err
	}

	// Mode
	if f.Mode != nil {
		if len(failModeWith) > 0 {
			return fmt.Errorf("'mode' can not be set with '%s'", failModeWith)
		}
		if *f.Mode&(^types.FileModeBitsMask) > 0 {
			return fmt.Errorf("'mode' does not match mask %#o: %#o", types.FileModeBitsMask, *f.Mode)
		}
	}

	// User / Uid
	if f.User != nil && f.Uid != nil {
		return fmt.Errorf("either 'user' or 'uid' can be set")
	}

	// Group / Gid
	if f.Group != nil && f.Gid != nil {
		return fmt.Errorf("either 'group' or 'gid' can be set")
	}

	return nil
}

func (f *File) Merge(otherResource Resource) error {
	panic("TODO")
}

func (f *File) Apply(ctx context.Context, host types.Host) error {
	panic("TODO")
}

// func (f *File) removeRecursively(ctx context.Context, host types.Host) error {
// 	err := host.Remove(ctx, f.Path)
// 	if err != nil {
// 		var errno syscall.Errno
// 		if errors.As(err, &errno) {
// 			switch errno {
// 			case syscall.ENOENT:
// 				return nil
// 			case syscall.ENOTEMPTY:
// 				break
// 			default:
// 				return err
// 			}
// 		} else {
// 			return err
// 		}
// 	}

// 	dirEntResultCh, cancel := host.ReadDir(ctx, f.Path)
// 	defer cancel()
// 	for dirEntResult := range dirEntResultCh {
// 		if dirEntResult.Error != nil {
// 			return err
// 		}
// 		subFile := File{Path: filepath.Join(f.Path, dirEntResult.DirEnt.Name), Absent: true}
// 		if err := subFile.Apply(ctx, host); err != nil {
// 			return err
// 		}
// 	}

// 	return host.Remove(ctx, f.Path)
// }

// func (f *File) applySocket(ctx context.Context, host types.Host, currentFile *File) error {
// 	if f.Socket {
// 		if !currentFile.Socket {
// 			if err := currentFile.removeRecursively(ctx, host); err != nil {
// 				return err
// 			}
// 			if err := host.Mknod(ctx, f.Path, *f.Mode|syscall.S_IFSOCK, 0); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

// func (f *File) applySymbolicLink(ctx context.Context, host types.Host, currentFile *File) error {
// 	if f.SymbolicLink != "" {
// 		if currentFile.SymbolicLink != f.SymbolicLink {
// 			if err := currentFile.removeRecursively(ctx, host); err != nil {
// 				return err
// 			}
// 			if err := host.Symlink(ctx, f.SymbolicLink, f.Path); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

// func (f *File) applyRegularFile(ctx context.Context, host types.Host, currentFile *File) error {
// 	if f.RegularFile != nil {
// 		if currentFile.RegularFile == nil || *currentFile.RegularFile != *f.RegularFile {
// 			if err := currentFile.removeRecursively(ctx, host); err != nil {
// 				return err
// 			}
// 			if err := host.WriteFile(ctx, string(f.Path), strings.NewReader(*f.RegularFile), *f.Mode); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

// func (f *File) applyBlockDevice(ctx context.Context, host types.Host, currentFile *File) error {
// 	if f.BlockDevice != nil {
// 		if currentFile.BlockDevice == nil || *currentFile.BlockDevice != *f.BlockDevice {
// 			if err := currentFile.removeRecursively(ctx, host); err != nil {
// 				return err
// 			}
// 			if err := host.Mknod(ctx, f.Path, *f.Mode|syscall.S_IFBLK, *f.BlockDevice); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

// func (f *File) applyDirectory(ctx context.Context, host types.Host, currentFile *File) error {
// 	if f.Directory != nil {
// 		if currentFile.Directory == nil {
// 			if err := currentFile.removeRecursively(ctx, host); err != nil {
// 				return err
// 			}
// 			if err := host.Mkdir(ctx, f.Path, *f.Mode); err != nil {
// 				return err
// 			}
// 		}
// 		pathToDelete := map[string]bool{}
// 		if currentFile.Directory != nil {
// 			for _, subFile := range *currentFile.Directory {
// 				pathToDelete[subFile.Path] = true
// 			}
// 		}
// 		for _, subFile := range *f.Directory {
// 			if err := subFile.Apply(ctx, host); err != nil {
// 				return err
// 			}
// 			delete(pathToDelete, subFile.Path)
// 		}
// 		for path := range pathToDelete {
// 			file := File{
// 				Path:   path,
// 				Absent: true,
// 			}
// 			if err := file.Apply(ctx, host); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

// func (f *File) applyCharacterDevice(ctx context.Context, host types.Host, currentFile *File) error {
// 	if f.CharacterDevice != nil {
// 		if currentFile.CharacterDevice == nil || *currentFile.CharacterDevice != *f.CharacterDevice {
// 			if err := currentFile.removeRecursively(ctx, host); err != nil {
// 				return err
// 			}
// 			if err := host.Mknod(ctx, f.Path, *f.Mode|syscall.S_IFCHR, *f.CharacterDevice); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

// func (f *File) applyFIFO(ctx context.Context, host types.Host, currentFile *File) error {
// 	if f.FIFO {
// 		if !currentFile.FIFO {
// 			if err := currentFile.removeRecursively(ctx, host); err != nil {
// 				return err
// 			}
// 			if err := host.Mknod(ctx, f.Path, *f.Mode|syscall.S_IFIFO, 0); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

// func (f *File) Apply(ctx context.Context, host types.Host) error {
// 	currentFile := &File{
// 		Path: f.Path,
// 	}
// 	if err := currentFile.Load(ctx, host); err != nil {
// 		return err
// 	}

// 	if f.Absent {
// 		return currentFile.removeRecursively(ctx, host)
// 	}

// 	if err := f.applySocket(ctx, host, currentFile); err != nil {
// 		return err
// 	}

// 	if err := f.applySymbolicLink(ctx, host, currentFile); err != nil {
// 		return err
// 	}

// 	if err := f.applyRegularFile(ctx, host, currentFile); err != nil {
// 		return err
// 	}

// 	if err := f.applyBlockDevice(ctx, host, currentFile); err != nil {
// 		return err
// 	}

// 	if err := f.applyDirectory(ctx, host, currentFile); err != nil {
// 		return err
// 	}

// 	if err := f.applyCharacterDevice(ctx, host, currentFile); err != nil {
// 		return err
// 	}

// 	if err := f.applyFIFO(ctx, host, currentFile); err != nil {
// 		return err
// 	}

// 	if f.Mode != nil {
// 		if err := host.Chmod(ctx, f.Path, *f.Mode); err != nil {
// 			return err
// 		}
// 	}

// 	if err := host.Lchown(ctx, f.Path, *f.Uid, *f.Gid); err != nil {
// 		return err
// 	}

// 	return nil
// }
