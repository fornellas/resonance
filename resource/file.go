package resource

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"gopkg.in/yaml.v3"

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

func (fp *FileState) Validate() error {
	if fp.Perm == os.FileMode(0) {
		return fmt.Errorf("missing 'perm'")
	}
	if fp.Uid != 0 && fp.User != "" {
		return fmt.Errorf("can't set both 'uid' and 'user'")
	}
	if fp.Gid != 0 && fp.Group != "" {
		return fmt.Errorf("can't set both 'gid' and 'group'")
	}
	return nil
}

func (fp *FileState) UnmarshalYAML(node *yaml.Node) error {
	type FileStateDecode FileState
	var fileStateDecode FileStateDecode
	node.KnownFields(true)
	if err := node.Decode(&fileStateDecode); err != nil {
		return err
	}
	fileState := FileState(fileStateDecode)
	if err := fileState.Validate(); err != nil {
		return fmt.Errorf("yaml line %d: validation error: %w", node.Line, err)
	}
	*fp = fileState
	return nil
}

func (fp FileState) GetUid(ctx context.Context, hst host.Host) (uint32, error) {
	if fp.User != "" {
		usr, err := hst.Lookup(ctx, fp.User)
		if err != nil {
			return 0, err
		}
		uid, err := strconv.ParseUint(usr.Uid, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("failed to parse UID: %s", usr.Uid)
		}
		return uint32(uid), nil
	}
	return fp.Uid, nil
}

func (fp FileState) GetGid(ctx context.Context, hst host.Host) (uint32, error) {
	if fp.Group != "" {
		group, err := hst.LookupGroup(ctx, fp.Group)
		if err != nil {
			return 0, err
		}
		gid, err := strconv.ParseUint(group.Gid, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("failed to parse GID: %s", group.Gid)
		}
		return uint32(gid), nil
	}
	return fp.Uid, nil
}

// File resource manages files.
type File struct{}

func (f File) Validate(name Name) error {
	path := string(name)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}
	return nil
}

func (f File) Check(ctx context.Context, hst host.Host, name Name, state State) (CheckResult, error) {
	logger := log.GetLogger(ctx)

	path := string(name)

	// FileState
	fileState := state.(*FileState)

	checkResult := CheckResult(true)

	// Path Hash
	pathtHash := md5.New()
	content, err := hst.ReadFile(ctx, path)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		logger.Debug("File not found")
		return false, nil
	} else {
		n, err := pathtHash.Write(content)
		if err != nil {
			return false, err
		}
		if n != len(content) {
			return false, fmt.Errorf("unexpected write length when generating md5: expected %d, got %d", len(content), n)
		}
	}

	// Instance Hash
	fileStateHash := md5.New()
	n, err := fileStateHash.Write([]byte(fileState.Content))
	if err != nil {
		return false, err
	}
	if n != len(fileState.Content) {
		return false, fmt.Errorf("unexpected write length when generating md5: expected %d, got %d", len(fileState.Content), n)
	}

	// Compare Hash
	if fmt.Sprintf("%v", pathtHash.Sum(nil)) != fmt.Sprintf("%v", fileStateHash.Sum(nil)) {
		logger.Debug("Content differs")
		checkResult = false
	}

	// FileInfo
	fileInfo, err := hst.Lstat(ctx, path)
	if err != nil {
		return false, err
	}
	stat_t := fileInfo.Sys().(*syscall.Stat_t)

	// Perm
	if fileInfo.Mode() != fileState.Perm {
		logger.Debugf("Expected permission 0%o, got 0%o", fileState.Perm, fileInfo.Mode())
		checkResult = false
	}

	// Uid / User
	uid, err := fileState.GetUid(ctx, hst)
	if err != nil {
		return false, err
	}
	if stat_t.Uid != uid {
		logger.Debugf("Expected UID %d, got %d", uid, stat_t.Uid)
		checkResult = false
	}

	// Gid
	gid, err := fileState.GetGid(ctx, hst)
	if err != nil {
		return false, err
	}
	if stat_t.Gid != gid {
		logger.Debugf("Expected GID %d, got %d", gid, stat_t.Gid)
		checkResult = false
	}

	return checkResult, nil
}

func (f File) Apply(ctx context.Context, hst host.Host, name Name, state State) error {
	nestedCtx := log.IndentLogger(ctx)
	path := string(name)

	// FileState
	fileState := state.(*FileState)

	// Content
	if err := hst.WriteFile(nestedCtx, path, []byte(fileState.Content), fileState.Perm); err != nil {
		return err
	}

	// Perm
	if err := hst.Chmod(nestedCtx, path, fileState.Perm); err != nil {
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
