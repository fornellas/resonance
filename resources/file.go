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

	"github.com/hashicorp/hcl/v2"

	"github.com/fornellas/resonance/host/types"
)

// File manages files
type File struct {
	// SourceLocations contains all locations in configuration files where this resource was defined
	SourceLocations []hcl.Range `hcl:",def_range"`
	// Path is the absolute path to the file
	Path string `hcl:"path,attr"`
	// Whether to remove the file
	Absent bool `hcl:"absent,optional"`
	// Create a socket file
	Socket bool `hcl:"socket,optional"`
	// Create a symbolic link pointing to given path
	SymbolicLink string `hcl:"symbolic_link,optional"`
	// Create a regular file with given contents
	RegularFile *string `hcl:"regular_file,optional"`
	// Create a block device file with given majon / minor.
	BlockDevice *types.FileDevice `hcl:"block_device,optional"`
	// Create a directory with given contents
	Directory []File `hcl:"directory,block"`
	// Create a character device file with given majon / minor
	CharacterDevice *types.FileDevice `hcl:"character_device,optional"`
	// Create a FIFO file
	FIFO bool `hcl:"fifo,optional"`
	// Mode bits 07777, see inode(7).
	Mode *types.FileMode `hcl:"mode,optional"`
	// User ID owner of the file. Default: 0.
	Uid *uint32 `hcl:"uid,optional"`
	// User name owner of the file
	User *string `hcl:"user,optional"`
	// Group ID owner of the file. Default: 0.
	Gid *uint32 `hcl:"gid,optional"`
	// Group name owner of the file
	Group *string `hcl:"group,optional"`
}

// FormatSourceLocation returns a human-readable string describing where this resource was defined
func (f *File) FormatSourceLocation() string {
	return FormatSourceLocations(f.SourceLocations)
}

func (f *File) mergeBool(current *bool, otherVal bool, fieldName string, other *File) error {
	if *current && otherVal && *current != otherVal {
		return fmt.Errorf("conflicting %s: %s declares %v, %s declares %v",
			fieldName, f.FormatSourceLocation(), *current, other.FormatSourceLocation(), otherVal)
	}
	if !*current && otherVal {
		*current = otherVal
	}
	return nil
}

func (f *File) mergeStringPtr(current **string, other **string, fieldName string, otherFile *File) error {
	if *current != nil && *other != nil && **current != **other {
		return fmt.Errorf("conflicting %s: %s declares %q, %s declares %q",
			fieldName, f.FormatSourceLocation(), **current, otherFile.FormatSourceLocation(), **other)
	}
	if *current == nil && *other != nil {
		*current = *other
	}
	return nil
}

func (f *File) mergeUint32Ptr(current **uint32, other **uint32, fieldName string, otherFile *File) error {
	if *current != nil && *other != nil && **current != **other {
		return fmt.Errorf("conflicting %s: %s declares %d, %s declares %d",
			fieldName, f.FormatSourceLocation(), **current, otherFile.FormatSourceLocation(), **other)
	}
	if *current == nil && *other != nil {
		*current = *other
	}
	return nil
}

func (f *File) mergeFileModePtr(current **types.FileMode, other **types.FileMode, fieldName string, otherFile *File) error {
	if *current != nil && *other != nil && **current != **other {
		return fmt.Errorf("conflicting %s: %s declares %#o, %s declares %#o",
			fieldName, f.FormatSourceLocation(), **current, otherFile.FormatSourceLocation(), **other)
	}
	if *current == nil && *other != nil {
		*current = *other
	}
	return nil
}

func (f *File) mergeFileDevicePtr(current **types.FileDevice, other **types.FileDevice, fieldName string, otherFile *File) error {
	if *current != nil && *other != nil && **current != **other {
		return fmt.Errorf("conflicting %s: %s declares %v, %s declares %v",
			fieldName, f.FormatSourceLocation(), **current, otherFile.FormatSourceLocation(), **other)
	}
	if *current == nil && *other != nil {
		*current = *other
	}
	return nil
}

func (f *File) mergeString(current *string, otherVal string, fieldName string, other *File) error {
	if *current != "" && otherVal != "" && *current != otherVal {
		return fmt.Errorf("conflicting %s: %s declares %q, %s declares %q",
			fieldName, f.FormatSourceLocation(), *current, other.FormatSourceLocation(), otherVal)
	}
	if *current == "" && otherVal != "" {
		*current = otherVal
	}
	return nil
}

// Merge attempts to merge another File resource into this one
// Returns error if there are conflicting values
func (f *File) Merge(other *File) error {
	if f.Path != other.Path {
		return fmt.Errorf("cannot merge files with different paths: %s vs %s", f.Path, other.Path)
	}

	// Collect source locations
	f.SourceLocations = append(f.SourceLocations, other.SourceLocations...)

	return f.mergeAllFields(other)
}

func (f *File) mergeAllFields(other *File) error {
	mergeOps := []func() error{
		func() error { return f.mergeBool(&f.Absent, other.Absent, "absent", other) },
		func() error { return f.mergeBool(&f.Socket, other.Socket, "socket", other) },
		func() error { return f.mergeBool(&f.FIFO, other.FIFO, "fifo", other) },
		func() error { return f.mergeString(&f.SymbolicLink, other.SymbolicLink, "symbolic_link", other) },
		func() error { return f.mergeStringPtr(&f.RegularFile, &other.RegularFile, "regular_file", other) },
		func() error { return f.mergeFileDevicePtr(&f.BlockDevice, &other.BlockDevice, "block_device", other) },
		func() error {
			return f.mergeFileDevicePtr(&f.CharacterDevice, &other.CharacterDevice, "character_device", other)
		},
		func() error { return f.mergeDirectory(other) },
		func() error { return f.mergeFileModePtr(&f.Mode, &other.Mode, "mode", other) },
		func() error { return f.mergeUint32Ptr(&f.Uid, &other.Uid, "uid", other) },
		func() error { return f.mergeStringPtr(&f.User, &other.User, "user", other) },
		func() error { return f.mergeUint32Ptr(&f.Gid, &other.Gid, "gid", other) },
		func() error { return f.mergeStringPtr(&f.Group, &other.Group, "group", other) },
	}

	for _, op := range mergeOps {
		if err := op(); err != nil {
			return err
		}
	}
	return nil
}

func (f *File) mergeDirectory(other *File) error {
	if len(f.Directory) == 0 && len(other.Directory) > 0 {
		f.Directory = other.Directory
		return nil
	}

	if len(other.Directory) == 0 {
		return nil
	}

	// Both have directory contents, merge recursively
	fileMap := make(map[string]*File)
	for i := range f.Directory {
		fileMap[f.Directory[i].Path] = &f.Directory[i]
	}

	for _, otherFile := range other.Directory {
		if existingFile, exists := fileMap[otherFile.Path]; exists {
			if err := existingFile.Merge(&otherFile); err != nil {
				return err
			}
		} else {
			f.Directory = append(f.Directory, otherFile)
		}
	}

	return nil
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
	if len(f.Directory) > 0 {
		for _, subFile := range f.Directory {
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

func (f *File) loadSymbolicLink(ctx context.Context, hst types.Host) error {
	target, err := hst.Readlink(ctx, f.Path)
	if err != nil {
		return err
	}
	f.SymbolicLink = target
	f.Mode = nil
	return nil
}

func (f *File) loadRegularFile(ctx context.Context, hst types.Host) error {
	fileReadCloser, err := hst.ReadFile(ctx, string(f.Path))
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

func (f *File) loadDirectory(ctx context.Context, hst types.Host) error {
	dirEntResultCh, cancel := hst.ReadDir(ctx, f.Path)
	defer cancel()

	directory := []File{}
	for dirEntResult := range dirEntResultCh {
		if dirEntResult.Error != nil {
			return dirEntResult.Error
		}
		subFile := File{Path: filepath.Join(f.Path, dirEntResult.DirEnt.Name)}
		if err := subFile.Load(ctx, hst); err != nil {
			return err
		}
		directory = append(directory, subFile)
	}

	sort.SliceStable(directory, func(i, j int) bool {
		return directory[i].Path < directory[j].Path
	})

	f.Directory = directory
	return nil
}

func (f *File) Load(ctx context.Context, hst types.Host) error {
	*f = File{
		Path: f.Path,
	}

	stat_t, err := hst.Lstat(ctx, string(f.Path))
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
		if err := f.loadSymbolicLink(ctx, hst); err != nil {
			return err
		}
	case syscall.S_IFREG:
		if err := f.loadRegularFile(ctx, hst); err != nil {
			return err
		}
	case syscall.S_IFBLK:
		f.BlockDevice = (*types.FileDevice)(&stat_t.Rdev)
	case syscall.S_IFDIR:
		if err := f.loadDirectory(ctx, hst); err != nil {
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

func (f *File) Resolve(ctx context.Context, hst types.Host) error {
	if len(f.Directory) > 0 {
		sort.SliceStable(f.Directory, func(i int, j int) bool {
			return f.Directory[i].Path < f.Directory[j].Path
		})
		for i, subFile := range f.Directory {
			if err := subFile.Resolve(ctx, hst); err != nil {
				return err
			}
			f.Directory[i] = subFile
		}
	}

	if f.User != nil {
		usr, err := hst.Lookup(ctx, *f.User)
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
		group, err := hst.LookupGroup(ctx, *f.Group)
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

func (f *File) removeRecursively(ctx context.Context, hst types.Host) error {
	err := hst.Remove(ctx, f.Path)
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

	dirEntResultCh, cancel := hst.ReadDir(ctx, f.Path)
	defer cancel()
	for dirEntResult := range dirEntResultCh {
		if dirEntResult.Error != nil {
			return err
		}
		subFile := File{Path: filepath.Join(f.Path, dirEntResult.DirEnt.Name), Absent: true}
		if err := subFile.Apply(ctx, hst); err != nil {
			return err
		}
	}

	return hst.Remove(ctx, f.Path)
}

func (f *File) applySocket(ctx context.Context, hst types.Host, currentFile *File) error {
	if f.Socket {
		if !currentFile.Socket {
			if err := currentFile.removeRecursively(ctx, hst); err != nil {
				return err
			}
			if err := hst.Mknod(ctx, f.Path, *f.Mode|syscall.S_IFSOCK, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) applySymbolicLink(ctx context.Context, hst types.Host, currentFile *File) error {
	if f.SymbolicLink != "" {
		if currentFile.SymbolicLink != f.SymbolicLink {
			if err := currentFile.removeRecursively(ctx, hst); err != nil {
				return err
			}
			if err := hst.Symlink(ctx, f.SymbolicLink, f.Path); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) applyRegularFile(ctx context.Context, hst types.Host, currentFile *File) error {
	if f.RegularFile != nil {
		if currentFile.RegularFile == nil || *currentFile.RegularFile != *f.RegularFile {
			if err := currentFile.removeRecursively(ctx, hst); err != nil {
				return err
			}
			if err := hst.WriteFile(ctx, string(f.Path), strings.NewReader(*f.RegularFile), *f.Mode); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) applyBlockDevice(ctx context.Context, hst types.Host, currentFile *File) error {
	if f.BlockDevice != nil {
		if currentFile.BlockDevice == nil || *currentFile.BlockDevice != *f.BlockDevice {
			if err := currentFile.removeRecursively(ctx, hst); err != nil {
				return err
			}
			if err := hst.Mknod(ctx, f.Path, *f.Mode|syscall.S_IFBLK, *f.BlockDevice); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) applyDirectory(ctx context.Context, hst types.Host, currentFile *File) error {
	if len(f.Directory) > 0 {
		if len(currentFile.Directory) == 0 {
			if err := currentFile.removeRecursively(ctx, hst); err != nil {
				return err
			}
			if err := hst.Mkdir(ctx, f.Path, *f.Mode); err != nil {
				return err
			}
		}
		pathToDelete := map[string]bool{}
		if len(currentFile.Directory) > 0 {
			for _, subFile := range currentFile.Directory {
				pathToDelete[subFile.Path] = true
			}
		}
		for _, subFile := range f.Directory {
			if err := subFile.Apply(ctx, hst); err != nil {
				return err
			}
			delete(pathToDelete, subFile.Path)
		}
		for path := range pathToDelete {
			file := File{
				Path:   path,
				Absent: true,
			}
			if err := file.Apply(ctx, hst); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) applyCharacterDevice(ctx context.Context, hst types.Host, currentFile *File) error {
	if f.CharacterDevice != nil {
		if currentFile.CharacterDevice == nil || *currentFile.CharacterDevice != *f.CharacterDevice {
			if err := currentFile.removeRecursively(ctx, hst); err != nil {
				return err
			}
			if err := hst.Mknod(ctx, f.Path, *f.Mode|syscall.S_IFCHR, *f.CharacterDevice); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) applyFIFO(ctx context.Context, hst types.Host, currentFile *File) error {
	if f.FIFO {
		if !currentFile.FIFO {
			if err := currentFile.removeRecursively(ctx, hst); err != nil {
				return err
			}
			if err := hst.Mknod(ctx, f.Path, *f.Mode|syscall.S_IFIFO, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) Apply(ctx context.Context, hst types.Host) error {
	currentFile := &File{
		Path: f.Path,
	}
	if err := currentFile.Load(ctx, hst); err != nil {
		return err
	}

	if f.Absent {
		return currentFile.removeRecursively(ctx, hst)
	}

	if err := f.applySocket(ctx, hst, currentFile); err != nil {
		return err
	}

	if err := f.applySymbolicLink(ctx, hst, currentFile); err != nil {
		return err
	}

	if err := f.applyRegularFile(ctx, hst, currentFile); err != nil {
		return err
	}

	if err := f.applyBlockDevice(ctx, hst, currentFile); err != nil {
		return err
	}

	if err := f.applyDirectory(ctx, hst, currentFile); err != nil {
		return err
	}

	if err := f.applyCharacterDevice(ctx, hst, currentFile); err != nil {
		return err
	}

	if err := f.applyFIFO(ctx, hst, currentFile); err != nil {
		return err
	}

	if f.Mode != nil {
		if err := hst.Chmod(ctx, f.Path, *f.Mode); err != nil {
			return err
		}
	}

	if err := hst.Lchown(ctx, f.Path, *f.Uid, *f.Gid); err != nil {
		return err
	}

	return nil
}
