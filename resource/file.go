package resource

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
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

func (fds FileState) Validate() error {
	if fds.Perm == os.FileMode(0) {
		return fmt.Errorf("missing 'perm'")
	}
	if fds.Uid != 0 && fds.User != "" {
		return fmt.Errorf("can't set both 'uid' and 'user'")
	}
	if fds.Gid != 0 && fds.Group != "" {
		return fmt.Errorf("can't set both 'gid' and 'group'")
	}
	return nil
}

func (fds FileState) GetUid(ctx context.Context, hst host.Host) (uint32, error) {
	if fds.User != "" {
		usr, err := hst.Lookup(ctx, fds.User)
		if err != nil {
			return 0, err
		}
		uid, err := strconv.ParseUint(usr.Uid, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("failed to parse UID: %s", usr.Uid)
		}
		return uint32(uid), nil
	}
	return fds.Uid, nil
}

func (fds FileState) GetGid(ctx context.Context, hst host.Host) (uint32, error) {
	if fds.Group != "" {
		group, err := hst.LookupGroup(ctx, fds.Group)
		if err != nil {
			return 0, err
		}
		gid, err := strconv.ParseUint(group.Gid, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("failed to parse GID: %s", group.Gid)
		}
		return uint32(gid), nil
	}
	return fds.Uid, nil
}

// File resource manages files.
type File struct{}

func (f File) ValidateName(name Name) error {
	path := string(name)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}
	return nil
}

func (f File) GetState(ctx context.Context, hst host.Host, name Name) (State, error) {
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

	return &fileState, nil
}

func (f File) DiffStates(
	ctx context.Context, hst host.Host,
	desiredState State, currentState State,
) ([]diffmatchpatch.Diff, error) {
	diffs := []diffmatchpatch.Diff{}
	desiredFileState := desiredState.(*FileState)
	currentFileState := currentState.(*FileState)

	uid, err := desiredFileState.GetUid(ctx, hst)
	if err != nil {
		return nil, err
	}
	gid, err := desiredFileState.GetGid(ctx, hst)
	if err != nil {
		return nil, err
	}
	diffs = append(diffs, Diff(currentFileState, FileState{
		Content: desiredFileState.Content,
		Perm:    desiredFileState.Perm,
		Uid:     uid,
		Gid:     gid,
	})...)

	return diffs, nil
}

func (f File) Apply(
	ctx context.Context, hst host.Host, name Name, state State,
) error {
	path := string(name)

	// FileState
	fileState := state.(*FileState)

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
	uid, err := fileState.GetUid(ctx, hst)
	if err != nil {
		return err
	}
	gid, err := fileState.GetGid(ctx, hst)
	if err != nil {
		return err
	}
	if stat_t.Uid != uid || stat_t.Gid != gid {
		if err := hst.Chown(ctx, path, int(fileState.Uid), int(fileState.Gid)); err != nil {
			return err
		}
	}

	return nil
}

func (f File) Destroy(ctx context.Context, hst host.Host, name Name) error {
	nestedCtx := log.IndentLogger(ctx)
	path := string(name)
	err := hst.Remove(nestedCtx, path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func init() {
	IndividuallyManageableResourceTypeMap["File"] = File{}
	ManageableResourcesStateMap["File"] = FileState{}
}
