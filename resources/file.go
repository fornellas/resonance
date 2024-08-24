package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/fornellas/resonance/host"
)

// File manages files
type File struct {
	// Path is the absolute path to the file
	Path string `yaml:"path" resonance:"id"`
	// Whether to remove the file
	Remove bool `yaml:"remove,omitempty"`
	// Contents of the file
	Content string `yaml:"content,omitempty"`
	// File permissions
	Perm os.FileMode `yaml:"perm,omitempty"`
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

	if f.Uid != 0 && f.User != "" {
		return fmt.Errorf("can't set both 'uid' and 'user'")
	}

	if f.Gid != 0 && f.Group != "" {
		return fmt.Errorf("can't set both 'gid' and 'group'")
	}

	return nil
}

func (f *File) Name() string {
	return f.Path
}

func (f *File) Load(ctx context.Context, hst host.Host) error {
	*f = File{
		Path: f.Path,
	}

	// Content
	content, err := hst.ReadFile(ctx, f.Path)
	if err != nil {
		if os.IsNotExist(err) {
			f.Remove = true
			return nil
		}
		return err
	}
	f.Content = string(content)

	// FileInfo
	fileInfo, err := hst.Lstat(ctx, f.Path)
	if err != nil {
		return err
	}

	// Perm
	f.Perm = fileInfo.Mode

	// Uid
	f.Uid = fileInfo.Uid

	// Gid
	f.Gid = fileInfo.Gid

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
	if f.Remove {
		err := hst.Remove(ctx, f.Path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Content
	if err := hst.WriteFile(ctx, f.Path, []byte(f.Content), f.Perm); err != nil {
		return err
	}

	// Perm
	if err := hst.Chmod(ctx, f.Path, f.Perm); err != nil {
		return err
	}

	// FileInfo
	fileInfo, err := hst.Lstat(ctx, f.Path)
	if err != nil {
		return err
	}

	// Uid / Gid
	if fileInfo.Uid != f.Uid || fileInfo.Gid != f.Gid {
		if err := hst.Chown(ctx, f.Path, int(f.Uid), int(f.Gid)); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	RegisterSingleResource(reflect.TypeOf((*File)(nil)).Elem())
}
