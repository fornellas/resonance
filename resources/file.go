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

// FileState manages files
type FileState struct {
	// Path is the absolute path to the file
	Path string `yaml:"path"`
	// Whether to remove the file
	Absent bool `yaml:"absent,omitempty"`
	// Create a socket file
	Socket bool `yaml:"socket,omitempty"`
	// Create a symbolic link pointing to given path
	SymbolicLink string `yaml:"symbolic_link,omitempty"`
	// Create a regular file with given contents
	RegularFile *string `yaml:"regular_file,omitempty"`
	// Create a block device file with given majon / minor.
	BlockDevice *types.FileDevice `yaml:"block_device,omitempty"`
	// Create a directory with given contents
	Directory *[]*FileState `yaml:"directory,omitempty"`
	// Create a character device file with given majon / minor
	CharacterDevice *types.FileDevice `yaml:"character_device,omitempty"`
	// Create a FIFO file
	FIFO bool `yaml:"FIFO,omitempty"`
	// Mode bits 07777, see inode(7).
	Mode *types.FileMode `yaml:"mode,omitempty"`
	// User ID owner of the file. Default: 0.
	Uid *uint32 `yaml:"uid,omitempty"`
	// User name owner of the file
	User *string `yaml:"user,omitempty"`
	// Group ID owner of the file. Default: 0.
	Gid *uint32 `yaml:"gid,omitempty"`
	// Group name owner of the file
	Group *string `yaml:"group,omitempty"`
}

func (s *FileState) validatePath() error {
	if s.Path == "" {
		return fmt.Errorf("'path' must be set")
	}

	if !filepath.IsAbs(s.Path) {
		return fmt.Errorf("'path' must be absolute: %#v", s.Path)
	}

	cleanPath := filepath.Clean(s.Path)
	if cleanPath != s.Path {
		return fmt.Errorf("'path' must be clean: %#v should be %#v", s.Path, cleanPath)
	}

	return nil
}

func (s *FileState) validateAbsentAndType() error {
	fileTypes := []bool{
		s.Socket,
		s.SymbolicLink != "",
		s.RegularFile != nil,
		s.BlockDevice != nil,
		s.Directory != nil,
		s.CharacterDevice != nil,
		s.FIFO,
	}

	typeCount := 0
	for _, isSet := range fileTypes {
		if isSet {
			typeCount++
		}
	}

	if typeCount == 0 {
		if s.Absent {
			if s.Mode != nil {
				return fmt.Errorf("can not set 'mode' with absent")
			}
			if s.Uid != nil {
				return fmt.Errorf("can not set 'uid' with absent")
			}
			if s.User != nil {
				return fmt.Errorf("can not set 'user' with absent")
			}
			if s.Gid != nil {
				return fmt.Errorf("can not set 'gid' with absent")
			}
			if s.Group != nil {
				return fmt.Errorf("can not set 'group' with absent")
			}
		} else {
			return fmt.Errorf("one file type must be defined without 'absent'")
		}
	} else if typeCount == 1 {
		if s.Absent {
			return fmt.Errorf("can not set 'absent' and a file type at the same time")
		}
	} else {
		return fmt.Errorf("only one file type can be defined")
	}

	return nil
}

func (s *FileState) validateDirectory() error {
	if s.Directory != nil {
		for _, subFile := range *s.Directory {
			if filepath.Dir(subFile.Path) != s.Path {
				return fmt.Errorf("directory entry '%s' is not a subpath of '%s'", subFile.Path, s.Path)
			}
			if err := subFile.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *FileState) validateMode() error {
	if s.Mode != nil {
		if *s.Mode&(^types.FileModeBitsMask) > 0 {
			return fmt.Errorf("file mode does not match mask %#o: %#o", types.FileModeBitsMask, *s.Mode)
		}
	}
	return nil
}

func (s *FileState) Validate() error {
	// Path
	if err := s.validatePath(); err != nil {
		return err
	}

	// Absent / Type
	if err := s.validateAbsentAndType(); err != nil {
		return err
	}

	// SymbolicLink
	if len(s.SymbolicLink) != 0 && s.Mode != nil {
		return fmt.Errorf("can not set 'mode' with symlink")
	}

	// Directory
	if err := s.validateDirectory(); err != nil {
		return err
	}

	// Mode
	if err := s.validateMode(); err != nil {
		return err
	}

	if s.Uid != nil && s.User != nil {
		return fmt.Errorf("can't set both 'uid' and 'user': %d, %#v", *s.Uid, *s.User)
	}

	if s.Gid != nil && s.Group != nil {
		return fmt.Errorf("can't set both 'gid' and 'group': %d, %#v", *s.Gid, *s.Group)
	}

	return nil
}

type FileProvisioner struct {
	Host types.Host
}

func NewFileProvisioner(host types.Host) (*FileProvisioner, error) {
	return &FileProvisioner{
		Host: host,
	}, nil
}

func (p *FileProvisioner) loadSymbolicLink(ctx context.Context, fs *FileState) error {
	target, err := p.Host.Readlink(ctx, fs.Path)
	if err != nil {
		return err
	}
	fs.SymbolicLink = target
	fs.Mode = nil
	return nil
}

func (p *FileProvisioner) loadRegularFile(ctx context.Context, fs *FileState) error {
	fileReadCloser, err := p.Host.ReadFile(ctx, string(fs.Path))
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
	fs.RegularFile = new(string)
	*fs.RegularFile = string(contentBytes)
	return nil
}

func (p *FileProvisioner) loadDirectory(ctx context.Context, fs *FileState) error {
	dirEntResultCh, cancel := p.Host.ReadDir(ctx, fs.Path)
	defer cancel()

	directory := []*FileState{}
	fs.Directory = &directory
	for dirEntResult := range dirEntResultCh {
		if dirEntResult.Error != nil {
			return dirEntResult.Error
		}
		subFile := &FileState{Path: filepath.Join(fs.Path, dirEntResult.DirEnt.Name)}
		if err := p.Load(ctx, subFile); err != nil {
			return err
		}
		directory = append(directory, subFile)
	}

	sort.SliceStable(directory, func(i, j int) bool {
		return directory[i].Path < directory[j].Path
	})

	return nil
}

func (p *FileProvisioner) Load(ctx context.Context, fs *FileState) error {
	*fs = FileState{
		Path: fs.Path,
	}

	stat_t, err := p.Host.Lstat(ctx, fs.Path)
	if err != nil {
		if os.IsNotExist(err) {
			fs.Absent = true
			return nil
		}
		return err
	}

	var mode types.FileMode = types.FileMode(stat_t.Mode) & types.FileModeBitsMask
	fs.Mode = &mode
	fs.Uid = &stat_t.Uid
	fs.Gid = &stat_t.Gid

	switch stat_t.Mode & syscall.S_IFMT {
	case syscall.S_IFSOCK:
		fs.Socket = true
	case syscall.S_IFLNK:
		if err := p.loadSymbolicLink(ctx, fs); err != nil {
			return err
		}
	case syscall.S_IFREG:
		if err := p.loadRegularFile(ctx, fs); err != nil {
			return err
		}
	case syscall.S_IFBLK:
		fs.BlockDevice = (*types.FileDevice)(&stat_t.Rdev)
	case syscall.S_IFDIR:
		if err := p.loadDirectory(ctx, fs); err != nil {
			return err
		}
	case syscall.S_IFCHR:
		fs.CharacterDevice = (*types.FileDevice)(&stat_t.Rdev)
	case syscall.S_IFIFO:
		fs.FIFO = true
	default:
		panic(fmt.Sprintf("bug: unexpected stat_t.Mode: 0x%x", stat_t.Mode))
	}

	return nil
}

func (p *FileProvisioner) Resolve(ctx context.Context, fs *FileState) error {

	if fs.Directory != nil {
		sort.SliceStable(*fs.Directory, func(i int, j int) bool {
			return (*fs.Directory)[i].Path < (*fs.Directory)[j].Path
		})
		for i, subFile := range *fs.Directory {
			if err := p.Resolve(ctx, subFile); err != nil {
				return err
			}
			(*fs.Directory)[i] = subFile
		}
	}

	if fs.User != nil {
		usr, err := p.Host.Lookup(ctx, *fs.User)
		if err != nil {
			return err
		}
		uid, err := strconv.ParseUint(usr.Uid, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse UID: %s", usr.Uid)
		}
		uid32 := uint32(uid)
		fs.Uid = &uid32
		fs.User = nil
	}
	if fs.Uid == nil && !fs.Absent {
		fs.Uid = new(uint32)
	}

	if fs.Group != nil {
		group, err := p.Host.LookupGroup(ctx, *fs.Group)
		if err != nil {
			return err
		}
		gid, err := strconv.ParseUint(group.Gid, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse GID: %s", group.Gid)
		}
		gid32 := uint32(gid)
		fs.Gid = &gid32
		fs.Group = nil
	}
	if fs.Gid == nil && !fs.Absent {
		fs.Gid = new(uint32)
	}

	return nil
}

func (p *FileProvisioner) removeRecursively(ctx context.Context, path string) error {
	err := p.Host.Remove(ctx, path)
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

	dirEntResultCh, cancel := p.Host.ReadDir(ctx, path)
	defer cancel()
	for dirEntResult := range dirEntResultCh {
		if dirEntResult.Error != nil {
			return err
		}
		subFile := &FileState{
			Path:   filepath.Join(path, dirEntResult.DirEnt.Name),
			Absent: true,
		}
		if err := p.Apply(ctx, subFile); err != nil {
			return err
		}
	}

	return p.Host.Remove(ctx, path)
}

func (p *FileProvisioner) applySocket(
	ctx context.Context,
	currentState, targetState *FileState,
) error {
	if targetState.Socket {
		if !currentState.Socket {
			if err := p.removeRecursively(ctx, currentState.Path); err != nil {
				return err
			}
			if err := p.Host.Mknod(ctx, targetState.Path, *targetState.Mode|syscall.S_IFSOCK, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *FileProvisioner) applySymbolicLink(
	ctx context.Context,
	currentState, targetState *FileState,
) error {
	if targetState.SymbolicLink != "" {
		if currentState.SymbolicLink != targetState.SymbolicLink {
			if err := p.removeRecursively(ctx, currentState.Path); err != nil {
				return err
			}
			if err := p.Host.Symlink(ctx, targetState.SymbolicLink, targetState.Path); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *FileProvisioner) applyRegularFile(
	ctx context.Context,
	currentState, targetState *FileState,
) error {
	if targetState.RegularFile != nil {
		if currentState.RegularFile == nil || *currentState.RegularFile != *targetState.RegularFile {
			if err := p.removeRecursively(ctx, currentState.Path); err != nil {
				return err
			}
			if err := p.Host.WriteFile(ctx, string(targetState.Path), strings.NewReader(*targetState.RegularFile), *targetState.Mode); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *FileProvisioner) applyBlockDevice(
	ctx context.Context,
	currentState, targetState *FileState,
) error {
	if targetState.BlockDevice != nil {
		if currentState.BlockDevice == nil || *currentState.BlockDevice != *targetState.BlockDevice {
			if err := p.removeRecursively(ctx, currentState.Path); err != nil {
				return err
			}
			if err := p.Host.Mknod(ctx, targetState.Path, *targetState.Mode|syscall.S_IFBLK, *targetState.BlockDevice); err != nil {
				return nil
			}
		}
	}
	return nil
}

func (p *FileProvisioner) applyDirectory(
	ctx context.Context,
	currentState, targetState *FileState,
) error {
	if targetState.Directory != nil {
		if currentState.Directory == nil {
			if err := p.removeRecursively(ctx, currentState.Path); err != nil {
				return err
			}
			if err := p.Host.Mkdir(ctx, targetState.Path, *targetState.Mode); err != nil {
				return err
			}
		}
		pathToDelete := map[string]bool{}
		if currentState.Directory != nil {
			for _, subFile := range *currentState.Directory {
				pathToDelete[subFile.Path] = true
			}
		}
		for _, subFile := range *targetState.Directory {
			if err := p.Apply(ctx, subFile); err != nil {
				return err
			}
			delete(pathToDelete, subFile.Path)
		}
		for path := range pathToDelete {
			fileState := &FileState{
				Path:   path,
				Absent: true,
			}
			if err := p.Apply(ctx, fileState); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *FileProvisioner) applyCharacterDevice(
	ctx context.Context,
	currentState, targetState *FileState,
) error {
	if targetState.CharacterDevice != nil {
		if currentState.CharacterDevice == nil || *currentState.CharacterDevice != *targetState.CharacterDevice {
			if err := p.removeRecursively(ctx, currentState.Path); err != nil {
				return err
			}
			if err := p.Host.Mknod(ctx, targetState.Path, *targetState.Mode|syscall.S_IFCHR, *targetState.CharacterDevice); err != nil {
				return nil
			}
		}
	}
	return nil
}

func (p *FileProvisioner) applyFIFO(
	ctx context.Context,
	currentState, targetState *FileState,
) error {
	if targetState.FIFO {
		if !currentState.FIFO {
			if err := p.removeRecursively(ctx, currentState.Path); err != nil {
				return err
			}
			if err := p.Host.Mknod(ctx, targetState.Path, *targetState.Mode|syscall.S_IFIFO, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *FileProvisioner) Apply(ctx context.Context, targetState *FileState) error {
	currentFile := &FileState{
		Path: targetState.Path,
	}
	if err := p.Load(ctx, currentFile); err != nil {
		return err
	}

	if targetState.Absent {
		return p.removeRecursively(ctx, currentFile.Path)
	}

	if err := p.applySocket(ctx, currentFile, targetState); err != nil {
		return err
	}

	if err := p.applySymbolicLink(ctx, currentFile, targetState); err != nil {
		return err
	}

	if err := p.applyRegularFile(ctx, currentFile, targetState); err != nil {
		return err
	}

	if err := p.applyBlockDevice(ctx, currentFile, targetState); err != nil {
		return err
	}

	if err := p.applyDirectory(ctx, currentFile, targetState); err != nil {
		return err
	}

	if err := p.applyCharacterDevice(ctx, currentFile, targetState); err != nil {
		return err
	}

	if err := p.applyFIFO(ctx, currentFile, targetState); err != nil {
		return err
	}

	if targetState.Mode != nil {
		if err := p.Host.Chmod(ctx, targetState.Path, *targetState.Mode); err != nil {
			return err
		}
	}

	if err := p.Host.Lchown(ctx, targetState.Path, *targetState.Uid, *targetState.Gid); err != nil {
		return err
	}

	return nil
}
