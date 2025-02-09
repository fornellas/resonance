package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/fornellas/resonance/host"
)

// File manages files
type File struct {
	// Path is the absolute path to the file
	Path string `yaml:"path"`
	// Whether to remove the file
	Absent bool `yaml:"absent,omitempty"`
	// Contents of the file
	Content string `yaml:"content,omitempty"`
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

	if !filepath.IsAbs(string(f.Path)) {
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

func (f *File) Load(ctx context.Context, hst host.Host) error {
	*f = File{
		Path: f.Path,
	}

	// Content
	fileReadCloser, err := hst.ReadFile(ctx, string(f.Path))
	if err != nil {
		if os.IsNotExist(err) {
			f.Absent = true
			return nil
		}
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
	f.Content = string(contentBytes)

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
	if err := hst.WriteFile(ctx, string(f.Path), bytes.NewReader([]byte(f.Content)), f.Mode); err != nil {
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
		if err := hst.Lchown(ctx, string(f.Path), f.Uid, f.Gid); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	RegisterSingleResource(reflect.TypeOf((*File)(nil)).Elem())
}
