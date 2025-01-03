package resources

import (
	"context"
	"fmt"
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
	// TODO Directory  Giving an archive file of the contents (eg: .tar, .zip etc).
	// TODO Directory  Pointing to a URL, which contains the archive file.
	// TODO Directory  A Git repository checked out at a given commit hash.
	// Create a character device file with given majon / minor
	CharacterDevice *uint64 `yaml:"character_device,omitempty"`
	// Create a FIFO file
	FIFO bool `yaml:"FIFO,omitempty"`
	// Mode bits, see inode(7)
	Mode uint32 `yaml:"mode,omitempty"`
	// User ID owner of the file
	Uid uint32 `yaml:"uid,omitempty"`
	// User name owner of the file
	User string `yaml:"user,omitempty"`
	// Group ID owner of the file
	Gid uint32 `yaml:"gid,omitempty"`
	// Group name owner of the file
	Group string `yaml:"group,omitempty"`
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

	if f.Uid != 0 && f.User != "" {
		return fmt.Errorf("can't set both 'uid' and 'user': %d, %#v", f.Uid, f.User)
	}

	if f.Gid != 0 && f.Group != "" {
		return fmt.Errorf("can't set both 'gid' and 'group': %d, %#v", f.Gid, f.Group)
	}

	return nil
}

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
	f.Uid = stat_t.Uid
	f.Gid = stat_t.Gid

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
		content, err := hst.ReadFile(ctx, f.Path)
		if err != nil {
			return err
		}
		regularFile := string(content)
		f.RegularFile = &regularFile
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
	if f.User != "" {
		usr, err := hst.Lookup(ctx, f.User)
		if err != nil {
			return err
		}
		uid, err := strconv.ParseUint(usr.Uid, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse UID: %s", usr.Uid)
		}
		f.Uid = uint32(uid)
		f.User = ""
	}

	if f.Group != "" {
		group, err := hst.LookupGroup(ctx, f.Group)
		if err != nil {
			return err
		}
		gid, err := strconv.ParseUint(group.Gid, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse GID: %s", group.Gid)
		}
		f.Gid = uint32(gid)
		f.Group = ""
	}

	return nil
}

func (f *File) applyAbsent(ctx context.Context, hst host.Host) error {
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
	return hst.Remove(ctx, f.Path)
}

func (f *File) Apply(ctx context.Context, hst host.Host) error {
	// Remove
	if f.Absent {
		return f.applyAbsent(ctx, hst)
	}

	// Socket
	// if f.Socket {

	// }

	// SymbolicLink
	// if f.SymbolicLink != "" {

	// }

	// RegularFile
	if f.RegularFile != nil {
		if err := hst.WriteFile(ctx, string(f.Path), []byte(*f.RegularFile), f.Mode); err != nil {
			return err
		}
	}

	// BlockDevice
	// if f.BlockDevice != nil {

	// }

	// Directory
	// if f.Directory != nil {

	// }

	// CharacterDevice
	// if f.CharacterDevice != nil {

	// }

	// FIFO
	// if f.FIFO {

	// }

	// FileInfo
	fileInfo, err := hst.Lstat(ctx, f.Path)
	if err != nil {
		return err
	}

	// Mode
	// TODO confirm if fileInfo.Mode has just mode bits or not
	if fileInfo.Mode != f.Mode {
		if err := hst.Chmod(ctx, string(f.Path), f.Mode); err != nil {
			return err
		}
	}

	// Uid / Gid
	if fileInfo.Uid != f.Uid || fileInfo.Gid != f.Gid {
		if err := hst.Chown(ctx, f.Path, f.Uid, f.Gid); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	RegisterSingleResource(reflect.TypeOf((*File)(nil)).Elem())
}
