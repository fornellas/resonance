package resources

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"

	"github.com/fornellas/resonance/host"
)

// File manages files
type File struct {
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
	// Create a block device file with given majon / minor
	BlockDevice *uint64 `yaml:"block_device,omitempty"`
	// Create a directory with given contents
	Directory *[]File `yaml:"directory,omitempty"`
	// Create a character device file with given majon / minor
	CharacterDevice *uint64 `yaml:"character_device,omitempty"`
	// Create a FIFO file
	FIFO bool `yaml:"FIFO,omitempty"`
	// Mode bits, see inode(7)
	Mode uint32 `yaml:"mode,omitempty"`
	// User ID owner of the file. Default: 0.
	Uid *uint32 `yaml:"uid,omitempty"`
	// User name owner of the file
	User *string `yaml:"user,omitempty"`
	// Group ID owner of the file. Default: 0.
	Gid *uint32 `yaml:"gid,omitempty"`
	// Group name owner of the file
	Group *string `yaml:"group,omitempty"`
	// TODO hard link?
}

func (f *File) validateType() error {
	fileTypes := []bool{
		f.Socket,
		f.SymbolicLink != "",
		f.RegularFile != nil,
		f.BlockDevice != nil,
		f.Directory != nil,
		f.CharacterDevice != nil,
		f.FIFO,
	}

	count := 0
	for _, isSet := range fileTypes {
		if isSet {
			count++
		}
	}

	if count == 0 {
		if !f.Absent {
			return fmt.Errorf("one file type must be defined")
		}
	} else if count == 1 {
		if f.Absent {
			return fmt.Errorf("can not set absent and a file type at the same time")
		}
		if f.Directory != nil {
			for _, subFile := range *f.Directory {
				if !strings.HasPrefix(subFile.Path, f.Path) {
					return fmt.Errorf("directory entry '%s' is not a subpath of '%s'", subFile.Path, f.Path)
				}
				if err := subFile.Validate(); err != nil {
					return err
				}
			}
		}
	} else {
		return fmt.Errorf("only one file type can be defined")
	}

	return nil
}

func (f *File) Validate() error {
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

	if err := f.validateType(); err != nil {
		return err
	}

	if f.Socket {
		// TODO mode bits can not be set
	}

	if f.Uid != nil && f.User != nil {
		return fmt.Errorf("can't set both 'uid' and 'user': %d, %#v", *f.Uid, *f.User)
	}

	if f.Gid != nil && f.Group != nil {
		return fmt.Errorf("can't set both 'gid' and 'group': %d, %#v", *f.Gid, *f.Group)
	}

	return nil
}

//gocyclo:ignore
func (f *File) Load(ctx context.Context, hst host.Host) error {
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

	f.Mode = stat_t.Mode & 07777
	f.Uid = &stat_t.Uid
	f.Gid = &stat_t.Gid

	switch stat_t.Mode & syscall.S_IFMT {
	case syscall.S_IFSOCK:
		f.Socket = true
	case syscall.S_IFLNK:
		target, err := hst.Readlink(ctx, f.Path)
		if err != nil {
			return err
		}
		f.SymbolicLink = target
	case syscall.S_IFREG:
		fileReadCloser, err := hst.ReadFile(ctx, string(f.Path))
		if err != nil {
			return err
		}
		contentBytes, err := io.ReadAll(fileReadCloser)
		if err != nil {
			return err
		}
		f.RegularFile = new(string)
		*f.RegularFile = string(contentBytes)
	case syscall.S_IFBLK:
		f.BlockDevice = &stat_t.Rdev
	case syscall.S_IFDIR:
		entries, err := hst.ReadDir(ctx, f.Path)
		if err != nil {
			return err
		}
		directory := make([]File, len(entries))
		f.Directory = &directory
		for i, entry := range entries {
			subFile := File{Path: filepath.Join(f.Path, entry.Name)}
			if err := subFile.Load(ctx, hst); err != nil {
				return err
			}
			directory[i] = subFile
		}
	case syscall.S_IFCHR:
		f.CharacterDevice = &stat_t.Rdev
	case syscall.S_IFIFO:
		f.FIFO = true
	default:
		panic(fmt.Sprintf("bug: unexpected stat_t.Mode: 0x%x", stat_t.Mode))
	}

	return nil
}

func (f *File) Resolve(ctx context.Context, hst host.Host) error {
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
	if f.Uid == nil {
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
	if f.Gid == nil {
		f.Gid = new(uint32)
	}

	return nil
}

func (f *File) removeRecursively(ctx context.Context, hst host.Host) error {
	stat, err := hst.Lstat(ctx, f.Path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
		entries, err := hst.ReadDir(ctx, f.Path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			subFile := File{Path: filepath.Join(f.Path, entry.Name), Absent: true}
			if err := subFile.Apply(ctx, hst); err != nil {
				return err
			}
		}
	}
	// FIXME if socket, block device or char device, fail if !root, because we won't be able to
	// recreate them if deleted
	return hst.Remove(ctx, f.Path)
}

//gocyclo:ignore
func (f *File) Apply(ctx context.Context, hst host.Host) error {
	currentFile := &File{
		Path: f.Path,
	}
	if err := currentFile.Load(ctx, hst); err != nil {
		return err
	}

	if f.Absent {
		return currentFile.removeRecursively(ctx, hst)
	}

	if f.Socket {
		if !currentFile.Socket {
			if err := currentFile.removeRecursively(ctx, hst); err != nil {
				return err
			}
		}
		// TODO create socket
	} else if f.SymbolicLink != "" {
		if currentFile.SymbolicLink != f.SymbolicLink {
			if err := currentFile.removeRecursively(ctx, hst); err != nil {
				return err
			}
		}
		// TODO create symlink
	} else if f.RegularFile != nil {
		if currentFile.RegularFile == nil || *currentFile.RegularFile != *f.RegularFile {
			if err := currentFile.removeRecursively(ctx, hst); err != nil {
				return err
			}
		}
		if err := hst.WriteFile(ctx, string(f.Path), []byte(*f.RegularFile), f.Mode); err != nil {
			return err
		}
	} else if f.BlockDevice != nil {
		if currentFile.BlockDevice == nil || *currentFile.BlockDevice != *f.BlockDevice {
			if err := currentFile.removeRecursively(ctx, hst); err != nil {
				return err
			}
		}
		// TODO create block device
	} else if f.Directory != nil {
		// TODO create directory with contents
	} else if f.CharacterDevice != nil {
		// TODO create char device
	} else if f.FIFO {
		// TODO create FIFO
	} else {
		panic("bug: unexpected file definition")
	}

	// Mode
	if currentFile.Mode != f.Mode {
		if err := hst.Chmod(ctx, string(f.Path), f.Mode); err != nil {
			return err
		}
	}

	// Uid / Gid
	if err := hst.Chown(ctx, f.Path, *f.Uid, *f.Gid); err != nil {
		return err
	}

	return nil
}

func init() {
	RegisterSingleResource(reflect.TypeOf((*File)(nil)).Elem())
}
