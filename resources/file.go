package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

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
	RegularFile string `yaml:"regular_file,omitempty"`
	// Create a block device file with given majon / minor
	BlockDevice int `yaml:"block_device,omitempty"`
	// Create a directory with given contents
	Directory []File `yaml:"directory,omitempty"`
	// TODO Directory  Giving an archive file of the contents (eg: .tar, .zip etc).
	// TODO Directory  Pointing to a URL, which contains the archive file.
	// TODO Directory  A Git repository checked out at a given commit hash.
	// Create a character device file with given majon / minor
	CharacterDevice int `yaml:"character_device,omitempty"`
	// Create a FIFO file
	FIFO bool `yaml:"FIFO,omitempty"`
	// Mode bits
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

func (f *File) Validate() error {
	if f.Path == "" {
		return fmt.Errorf("'path' must be set")
	}

	if !filepath.IsAbs(f.Path) {
		return fmt.Errorf("'path' must be absolute")
	}

	if filepath.Clean(f.Path) != f.Path {
		return fmt.Errorf("'path' must be clean")
	}

	if f.Uid != 0 && f.User != "" {
		return fmt.Errorf("can't set both 'uid' and 'user'")
	}

	if f.Gid != 0 && f.Group != "" {
		return fmt.Errorf("can't set both 'gid' and 'group'")
	}

	fileTypes := []bool{
		f.Socket,
		f.SymbolicLink != "",
		f.RegularFile != "",
		f.BlockDevice != 0,
		len(f.Directory) > 0,
		f.CharacterDevice != 0,
		f.FIFO,
	}

	count := 0
	for _, isSet := range fileTypes {
		if isSet {
			count++
		}
	}

	if count > 1 {
		return fmt.Errorf("at most one file type must be defined")
	}

	if len(f.Directory) > 0 {
		for _, subFile := range f.Directory {
			if !strings.HasPrefix(subFile.Path, f.Path) {
				return fmt.Errorf("directory entry '%s' is not a subpath of '%s'", subFile.Path, f.Path)
			}
		}
	}

	return nil
}

func (f *File) Load(ctx context.Context, hst host.Host) error {
	*f = File{
		Path: f.Path,
	}

	// Content
	content, err := hst.ReadFile(ctx, string(f.Path))
	if err != nil {
		if os.IsNotExist(err) {
			f.Absent = true
			return nil
		}
		return err
	}
	f.RegularFile = string(content)

	// FileInfo
	stat_t, err := hst.Lstat(ctx, string(f.Path))
	if err != nil {
		return err
	}

	// Perm
	f.Mode = stat_t.Mode & 07777

	// Uid
	f.Uid = stat_t.Uid

	// Gid
	f.Gid = stat_t.Gid

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

func (f *File) Apply(ctx context.Context, hst host.Host) error {
	// Remove
	if f.Absent {
		err := hst.Remove(ctx, string(f.Path))
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Content
	if err := hst.WriteFile(ctx, string(f.Path), []byte(f.RegularFile), f.Mode); err != nil {
		return err
	}

	// Perm
	if err := hst.Chmod(ctx, string(f.Path), f.Mode); err != nil {
		return err
	}

	// FileInfo
	fileInfo, err := hst.Lstat(ctx, string(f.Path))
	if err != nil {
		return err
	}

	// Uid / Gid
	if fileInfo.Uid != f.Uid || fileInfo.Gid != f.Gid {
		if err := hst.Chown(ctx, string(f.Path), int(f.Uid), int(f.Gid)); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	RegisterSingleResource(reflect.TypeOf((*File)(nil)).Elem())
}
