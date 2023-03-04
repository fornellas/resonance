package resource

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

// FileParams for File
type FileParams struct {
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

func (fp *FileParams) Validate() error {
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

func (fp *FileParams) UnmarshalYAML(node *yaml.Node) error {
	type FileParamsDecode FileParams
	var fileParamsDecode FileParamsDecode
	if err := node.Decode(&fileParamsDecode); err != nil {
		return err
	}
	fileParams := FileParams(fileParamsDecode)
	if err := fileParams.Validate(); err != nil {
		return err
	}
	*fp = fileParams
	return nil
}

func (fp FileParams) GetUid(ctx context.Context, hst host.Host) (uint32, error) {
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

func (fp FileParams) GetGid(ctx context.Context, hst host.Host) (uint32, error) {
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

func (f File) Check(ctx context.Context, hst host.Host, name Name, parameters yaml.Node) (CheckResult, error) {
	logger := log.GetLogger(ctx)

	path := string(name)

	// FileParams
	var fileParams FileParams
	if err := parameters.Decode(&fileParams); err != nil {
		return false, err
	}

	checkResult := CheckResult(true)

	// Path Hash
	pathtHash := md5.New()
	content, err := hst.ReadFile(ctx, path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
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
	fileParamsHash := md5.New()
	n, err := fileParamsHash.Write([]byte(fileParams.Content))
	if err != nil {
		return false, err
	}
	if n != len(fileParams.Content) {
		return false, fmt.Errorf("unexpected write length when generating md5: expected %d, got %d", len(fileParams.Content), n)
	}

	// Compare Hash
	if fmt.Sprintf("%v", pathtHash.Sum(nil)) != fmt.Sprintf("%v", fileParamsHash.Sum(nil)) {
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
	if fileInfo.Mode() != fileParams.Perm {
		logger.Debugf("Expected permission 0%o, got 0%o", fileParams.Perm, fileInfo.Mode())
		checkResult = false
	}

	// Uid / User
	uid, err := fileParams.GetUid(ctx, hst)
	if err != nil {
		return false, err
	}
	if stat_t.Uid != uid {
		logger.Debugf("Expected UID %d, got %d", uid, stat_t.Uid)
		checkResult = false
	}

	// Gid
	gid, err := fileParams.GetGid(ctx, hst)
	if err != nil {
		return false, err
	}
	if stat_t.Gid != gid {
		logger.Debugf("Expected GID %d, got %d", gid, stat_t.Gid)
		checkResult = false
	}

	return checkResult, nil
}

func (f File) Refresh(ctx context.Context, hst host.Host, name Name) error {
	return nil
}

func (f File) Apply(ctx context.Context, hst host.Host, name Name, parameters yaml.Node) error {
	nestedCtx := log.IndentLogger(ctx)
	path := string(name)

	// FileParams
	var fileParams FileParams
	if err := parameters.Decode(&fileParams); err != nil {
		return err
	}

	// Content
	if err := hst.WriteFile(nestedCtx, path, []byte(fileParams.Content), fileParams.Perm); err != nil {
		return err
	}

	// Perm
	if err := hst.Chmod(nestedCtx, path, fileParams.Perm); err != nil {
		return err
	}

	// FileInfo
	fileInfo, err := hst.Lstat(ctx, path)
	if err != nil {
		return err
	}
	stat_t := fileInfo.Sys().(*syscall.Stat_t)

	// Uid / Gid
	uid, err := fileParams.GetUid(ctx, hst)
	if err != nil {
		return err
	}
	gid, err := fileParams.GetGid(ctx, hst)
	if err != nil {
		return err
	}
	if stat_t.Uid != uid || stat_t.Gid != gid {
		if err := hst.Chown(ctx, path, int(fileParams.Uid), int(fileParams.Gid)); err != nil {
			return err
		}
	}

	return nil
}

func (f File) Destroy(ctx context.Context, hst host.Host, name Name) error {
	nestedCtx := log.IndentLogger(ctx)
	path := string(name)
	return hst.Remove(nestedCtx, path)
}

func init() {
	IndividuallyManageableResourceTypeMap["File"] = File{}
}
