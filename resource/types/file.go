package resource

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"
)

// FileState for File
type FileState struct {
	// Contents of the file
	Content string `yaml:"content"`
	// File permissions
	Perm os.FileMode `yaml:"perm"`
	// User ID owner of the file
	Uid uint32 `yaml:"uid"`
	// User name owner of the file
	User string `yaml:"user"`
	// Group ID owner of the file
	Gid uint32 `yaml:"gid"`
	// Group name owner of the file
	Group string `yaml:"group"`
}

func (fs FileState) ValidateAndUpdate(ctx context.Context, hst host.Host) (resource.State, error) {
	// Validate
	if fs.Perm == os.FileMode(0) {
		return nil, fmt.Errorf("missing 'perm'")
	}
	if fs.Uid != 0 && fs.User != "" {
		return nil, fmt.Errorf("can't set both 'uid' and 'user'")
	}
	if fs.Gid != 0 && fs.Group != "" {
		return nil, fmt.Errorf("can't set both 'gid' and 'group'")
	}

	// Update
	if fs.User != "" {
		usr, err := hst.Lookup(ctx, fs.User)
		if err != nil {
			return nil, err
		}
		uid, err := strconv.ParseUint(usr.Uid, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UID: %s", usr.Uid)
		}
		fs.Uid = uint32(uid)
		fs.User = ""
	}
	if fs.Group != "" {
		group, err := hst.LookupGroup(ctx, fs.Group)
		if err != nil {
			return nil, err
		}
		gid, err := strconv.ParseUint(group.Gid, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GID: %s", group.Gid)
		}
		fs.Gid = uint32(gid)
		fs.Group = ""
	}

	return fs, nil
}

// File resource manages files.
type File struct{}

func (f File) ValidateName(name resource.Name) error {
	path := string(name)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}
	return nil
}

func (f File) GetState(ctx context.Context, hst host.Host, name resource.Name) (resource.State, error) {
	logger := log.GetLogger(ctx)

	path := string(name)

	fileState := FileState{}

	// Content
	content, err := hst.ReadFile(ctx, path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("File not found")
			return nil, nil
		}
		return nil, err
	}
	fileState.Content = string(content)

	// FileInfo
	fileInfo, err := hst.Lstat(ctx, path)
	if err != nil {
		return nil, err
	}
	stat_t := fileInfo.Sys().(*syscall.Stat_t)

	// Perm
	fileState.Perm = fileInfo.Mode()

	// Uid
	fileState.Uid = stat_t.Uid

	// Gid
	fileState.Gid = stat_t.Gid

	return fileState, nil
}

func (f File) Configure(
	ctx context.Context, hst host.Host, name resource.Name, state resource.State,
) error {
	path := string(name)

	// FileState
	fileState := state.(FileState)

	// Content
	if err := hst.WriteFile(ctx, path, []byte(fileState.Content), fileState.Perm); err != nil {
		return err
	}

	// Perm
	if err := hst.Chmod(ctx, path, fileState.Perm); err != nil {
		return err
	}

	// FileInfo
	fileInfo, err := hst.Lstat(ctx, path)
	if err != nil {
		return err
	}
	stat_t := fileInfo.Sys().(*syscall.Stat_t)

	// Uid / Gid
	if stat_t.Uid != fileState.Uid || stat_t.Gid != fileState.Gid {
		if err := hst.Chown(ctx, path, int(fileState.Uid), int(fileState.Gid)); err != nil {
			return err
		}
	}

	return nil
}

func (f File) Destroy(ctx context.Context, hst host.Host, name resource.Name) error {
	nestedCtx := log.IndentLogger(ctx)
	path := string(name)
	err := hst.Remove(nestedCtx, path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func init() {
	resource.IndividuallyManageableResourceTypeMap["File"] = File{}
	resource.ManageableResourcesStateMap["File"] = FileState{}
}
