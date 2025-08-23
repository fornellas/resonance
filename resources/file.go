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
	"strings"
	"syscall"

	"github.com/fornellas/resonance/host/types"
)

// File manages files
type File struct {
	// Path is the absolute path to the file
	Path string
	// Whether to remove the file
	Absent bool
	// Create a socket file
	Socket bool
	// Create a symbolic link pointing to given path
	SymbolicLink string
	// Create a regular file with given contents
	RegularFile *string
	// Create a block device file with given majon / minor.
	BlockDevice *types.FileDevice
	// Create a directory with given contents
	Directory *[]File
	// Create a character device file with given majon / minor
	CharacterDevice *types.FileDevice
	// Create a FIFO file
	FIFO bool
	// Mode bits 07777, see inode(7).
	Mode *types.FileMode
	// User ID owner of the file. Default: 0.
	Uid *uint32
	// User name owner of the file
	User *string
	// Group ID owner of the file. Default: 0.
	Gid *uint32
	// Group name owner of the file
	Group *string
}

func (f *File) validatePath() error {
	if f.Path == "" {
		return fmt.Errorf("'path' must be set")
	}

	if !filepath.IsAbs(f.Path) {
		return fmt.Errorf("'path' must be absolute: %#v", f.Path)
	}

	cleanPath := filepath.Clean(f.Path)
	if cleanPath != f.Path {
		return fmt.Errorf("'path' must be clean: %#v should be %#v", f.Path, cleanPath)
	}

	return nil
}

func (f *File) validateAbsentAndType() error {
	fileTypes := []bool{
		f.Socket,
		f.SymbolicLink != "",
		f.RegularFile != nil,
		f.BlockDevice != nil,
		f.Directory != nil,
		f.CharacterDevice != nil,
		f.FIFO,
	}

	typeCount := 0
	for _, isSet := range fileTypes {
		if isSet {
			typeCount++
		}
	}

	if typeCount == 0 {
		if f.Absent {
			if f.Mode != nil {
				return fmt.Errorf("can not set 'mode' with absent")
			}
			if f.Uid != nil {
				return fmt.Errorf("can not set 'uid' with absent")
			}
			if f.User != nil {
				return fmt.Errorf("can not set 'user' with absent")
			}
			if f.Gid != nil {
				return fmt.Errorf("can not set 'gid' with absent")
			}
			if f.Group != nil {
				return fmt.Errorf("can not set 'group' with absent")
			}
		} else {
			return fmt.Errorf("one file type must be defined without 'absent'")
		}
	} else if typeCount == 1 {
		if f.Absent {
			return fmt.Errorf("can not set 'absent' and a file type at the same time")
		}
	} else {
		return fmt.Errorf("only one file type can be defined")
	}

	return nil
}

func (f *File) validateDirectory() error {
	if f.Directory != nil {
		for _, subFile := range *f.Directory {
			if filepath.Dir(subFile.Path) != f.Path {
				return fmt.Errorf("directory entry '%s' is not a subpath of '%s'", subFile.Path, f.Path)
			}
			if err := subFile.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) validateMode() error {
	if f.Mode != nil {
		if *f.Mode&(^types.FileModeBitsMask) > 0 {
			return fmt.Errorf("file mode does not match mask %#o: %#o", types.FileModeBitsMask, *f.Mode)
		}
	}
	return nil
}

func (f *File) Validate() error {
	// Path
	if err := f.validatePath(); err != nil {
		return err
	}

	// Absent / Type
	if err := f.validateAbsentAndType(); err != nil {
		return err
	}

	// SymbolicLink
	if len(f.SymbolicLink) != 0 && f.Mode != nil {
		return fmt.Errorf("can not set 'mode' with symlink")
	}

	// Directory
	if err := f.validateDirectory(); err != nil {
		return err
	}

	// Mode
	if err := f.validateMode(); err != nil {
		return err
	}

	if f.Uid != nil && f.User != nil {
		return fmt.Errorf("can't set both 'uid' and 'user': %d, %#v", *f.Uid, *f.User)
	}

	if f.Gid != nil && f.Group != nil {
		return fmt.Errorf("can't set both 'gid' and 'group': %d, %#v", *f.Gid, *f.Group)
	}

	return nil
}

func (f *File) loadSymbolicLink(ctx context.Context, host types.Host) error {
	target, err := host.Readlink(ctx, f.Path)
	if err != nil {
		return err
	}
	f.SymbolicLink = target
	f.Mode = nil
	return nil
}

func (f *File) loadRegularFile(ctx context.Context, host types.Host) error {
	fileReadCloser, err := host.ReadFile(ctx, string(f.Path))
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
	f.RegularFile = new(string)
	*f.RegularFile = string(contentBytes)
	return nil
}

func (f *File) loadDirectory(ctx context.Context, host types.Host) error {
	dirEntResultCh, cancel := host.ReadDir(ctx, f.Path)
	defer cancel()

	directory := []File{}
	f.Directory = &directory
	for dirEntResult := range dirEntResultCh {
		if dirEntResult.Error != nil {
			return dirEntResult.Error
		}
		subFile := File{Path: filepath.Join(f.Path, dirEntResult.DirEnt.Name)}
		if err := subFile.Load(ctx, host); err != nil {
			return err
		}
		directory = append(directory, subFile)
	}

	sort.SliceStable(directory, func(i, j int) bool {
		return directory[i].Path < directory[j].Path
	})

	return nil
}

func (f *File) Load(ctx context.Context, host types.Host) error {
	*f = File{
		Path: f.Path,
	}

	stat_t, err := host.Lstat(ctx, string(f.Path))
	if err != nil {
		if os.IsNotExist(err) {
			f.Absent = true
			return nil
		}
		return err
	}

	var mode types.FileMode = types.FileMode(stat_t.Mode) & types.FileModeBitsMask
	f.Mode = &mode
	f.Uid = &stat_t.Uid
	f.Gid = &stat_t.Gid

	switch stat_t.Mode & syscall.S_IFMT {
	case syscall.S_IFSOCK:
		f.Socket = true
	case syscall.S_IFLNK:
		if err := f.loadSymbolicLink(ctx, host); err != nil {
			return err
		}
	case syscall.S_IFREG:
		if err := f.loadRegularFile(ctx, host); err != nil {
			return err
		}
	case syscall.S_IFBLK:
		f.BlockDevice = (*types.FileDevice)(&stat_t.Rdev)
	case syscall.S_IFDIR:
		if err := f.loadDirectory(ctx, host); err != nil {
			return err
		}
	case syscall.S_IFCHR:
		f.CharacterDevice = (*types.FileDevice)(&stat_t.Rdev)
	case syscall.S_IFIFO:
		f.FIFO = true
	default:
		panic(fmt.Sprintf("bug: unexpected stat_t.Mode: 0x%x", stat_t.Mode))
	}

	return nil
}

func (f *File) Resolve(ctx context.Context, host types.Host) error {
	if f.Directory != nil {
		sort.SliceStable(*f.Directory, func(i int, j int) bool {
			return (*f.Directory)[i].Path < (*f.Directory)[j].Path
		})
		for i, subFile := range *f.Directory {
			if err := subFile.Resolve(ctx, host); err != nil {
				return err
			}
			(*f.Directory)[i] = subFile
		}
	}

	if f.User != nil {
		usr, err := host.Lookup(ctx, *f.User)
		if err != nil {
			return err
		}
		uid, err := strconv.ParseUint(usr.Uid, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse UID: %s", usr.Uid)
		}
		uid32 := uint32(uid)
		f.Uid = &uid32
		f.User = nil
	}
	if f.Uid == nil && !f.Absent {
		f.Uid = new(uint32)
	}

	if f.Group != nil {
		group, err := host.LookupGroup(ctx, *f.Group)
		if err != nil {
			return err
		}
		gid, err := strconv.ParseUint(group.Gid, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse GID: %s", group.Gid)
		}
		gid32 := uint32(gid)
		f.Gid = &gid32
		f.Group = nil
	}
	if f.Gid == nil && !f.Absent {
		f.Gid = new(uint32)
	}

	return nil
}

func (f *File) removeRecursively(ctx context.Context, host types.Host) error {
	err := host.Remove(ctx, f.Path)
	if err != nil {
		var errno syscall.Errno
		if errors.As(err, &errno) {
			switch errno {
			case syscall.ENOENT:
				return nil
			case syscall.ENOTEMPTY:
				break
			default:
				return err
			}
		} else {
			return err
		}
	}

	dirEntResultCh, cancel := host.ReadDir(ctx, f.Path)
	defer cancel()
	for dirEntResult := range dirEntResultCh {
		if dirEntResult.Error != nil {
			return err
		}
		subFile := File{Path: filepath.Join(f.Path, dirEntResult.DirEnt.Name), Absent: true}
		if err := subFile.Apply(ctx, host); err != nil {
			return err
		}
	}

	return host.Remove(ctx, f.Path)
}

func (f *File) applySocket(ctx context.Context, host types.Host, currentFile *File) error {
	if f.Socket {
		if !currentFile.Socket {
			if err := currentFile.removeRecursively(ctx, host); err != nil {
				return err
			}
			if err := host.Mknod(ctx, f.Path, *f.Mode|syscall.S_IFSOCK, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) applySymbolicLink(ctx context.Context, host types.Host, currentFile *File) error {
	if f.SymbolicLink != "" {
		if currentFile.SymbolicLink != f.SymbolicLink {
			if err := currentFile.removeRecursively(ctx, host); err != nil {
				return err
			}
			if err := host.Symlink(ctx, f.SymbolicLink, f.Path); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) applyRegularFile(ctx context.Context, host types.Host, currentFile *File) error {
	if f.RegularFile != nil {
		if currentFile.RegularFile == nil || *currentFile.RegularFile != *f.RegularFile {
			if err := currentFile.removeRecursively(ctx, host); err != nil {
				return err
			}
			if err := host.WriteFile(ctx, string(f.Path), strings.NewReader(*f.RegularFile), *f.Mode); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) applyBlockDevice(ctx context.Context, host types.Host, currentFile *File) error {
	if f.BlockDevice != nil {
		if currentFile.BlockDevice == nil || *currentFile.BlockDevice != *f.BlockDevice {
			if err := currentFile.removeRecursively(ctx, host); err != nil {
				return err
			}
			if err := host.Mknod(ctx, f.Path, *f.Mode|syscall.S_IFBLK, *f.BlockDevice); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) applyDirectory(ctx context.Context, host types.Host, currentFile *File) error {
	if f.Directory != nil {
		if currentFile.Directory == nil {
			if err := currentFile.removeRecursively(ctx, host); err != nil {
				return err
			}
			if err := host.Mkdir(ctx, f.Path, *f.Mode); err != nil {
				return err
			}
		}
		pathToDelete := map[string]bool{}
		if currentFile.Directory != nil {
			for _, subFile := range *currentFile.Directory {
				pathToDelete[subFile.Path] = true
			}
		}
		for _, subFile := range *f.Directory {
			if err := subFile.Apply(ctx, host); err != nil {
				return err
			}
			delete(pathToDelete, subFile.Path)
		}
		for path := range pathToDelete {
			file := File{
				Path:   path,
				Absent: true,
			}
			if err := file.Apply(ctx, host); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) applyCharacterDevice(ctx context.Context, host types.Host, currentFile *File) error {
	if f.CharacterDevice != nil {
		if currentFile.CharacterDevice == nil || *currentFile.CharacterDevice != *f.CharacterDevice {
			if err := currentFile.removeRecursively(ctx, host); err != nil {
				return err
			}
			if err := host.Mknod(ctx, f.Path, *f.Mode|syscall.S_IFCHR, *f.CharacterDevice); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) applyFIFO(ctx context.Context, host types.Host, currentFile *File) error {
	if f.FIFO {
		if !currentFile.FIFO {
			if err := currentFile.removeRecursively(ctx, host); err != nil {
				return err
			}
			if err := host.Mknod(ctx, f.Path, *f.Mode|syscall.S_IFIFO, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) Apply(ctx context.Context, host types.Host) error {
	currentFile := &File{
		Path: f.Path,
	}
	if err := currentFile.Load(ctx, host); err != nil {
		return err
	}

	if f.Absent {
		return currentFile.removeRecursively(ctx, host)
	}

	if err := f.applySocket(ctx, host, currentFile); err != nil {
		return err
	}

	if err := f.applySymbolicLink(ctx, host, currentFile); err != nil {
		return err
	}

	if err := f.applyRegularFile(ctx, host, currentFile); err != nil {
		return err
	}

	if err := f.applyBlockDevice(ctx, host, currentFile); err != nil {
		return err
	}

	if err := f.applyDirectory(ctx, host, currentFile); err != nil {
		return err
	}

	if err := f.applyCharacterDevice(ctx, host, currentFile); err != nil {
		return err
	}

	if err := f.applyFIFO(ctx, host, currentFile); err != nil {
		return err
	}

	if f.Mode != nil {
		if err := host.Chmod(ctx, f.Path, *f.Mode); err != nil {
			return err
		}
	}

	if err := host.Lchown(ctx, f.Path, *f.Uid, *f.Gid); err != nil {
		return err
	}

	return nil
}
