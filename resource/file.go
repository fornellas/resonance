package resource

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io/fs"
	"os"
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
	// User string `yaml:"user"`
	// Group ID owner of the file
	Gid uint32 `yaml:"gid"`
	// Group name owner of the file
	// Group string `yaml:"group"`
}

// File resource manages files.
type File struct{}

func (f File) Check(ctx context.Context, hst host.Host, name Name, parameters yaml.Node) (CheckResult, error) {
	logger := log.GetLogger(ctx)

	path := string(name)

	// FileParams
	var fileParams FileParams
	if err := parameters.Decode(&fileParams); err != nil {
		return false, err
	}

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
		logger.Debug("Hash differs")
		return false, nil
	}

	// Perm
	fileInfo, err := hst.Lstat(ctx, path)
	if err != nil {
		return false, err
	}
	if fileInfo.Mode() != fileParams.Perm {
		logger.Debug("Perm differs")
		return false, nil
	}

	// Uid
	if fileInfo.Sys().(*syscall.Stat_t).Uid != fileParams.Uid {
		logger.Debug("Uid differs")
		return false, nil
	}

	// User
	// TODO use host interface
	// u, err := user.LookupId(strconv.Itoa(int(fileState.Uid)))
	// if err != nil {
	// 	return false, err
	// }
	// fileState.User = u.Username

	// Gid
	if fileInfo.Sys().(*syscall.Stat_t).Gid != fileParams.Gid {
		logger.Debug("Gid differs")
		return false, nil
	}

	// Group
	// TODO use host interface
	// g, err := user.LookupGroupId(strconv.Itoa(int(fileState.Gid)))
	// if err != nil {
	// 	return false, err
	// }
	// fileState.Group = g.Name

	return true, nil
}

func (f File) Refresh(ctx context.Context, hst host.Host, name Name) error {
	return nil
}

func (f File) Apply(ctx context.Context, hst host.Host, name Name, parameters yaml.Node) error {
	// TODO use Host interface
	// fileParams := parameters.(FileParams)

	// if err := os.WriteFile(name, fileParams.Content, fileParams.Perm); err != nil {
	// 	return err
	// }
	// return nil
	return fmt.Errorf("TODO File.Apply")
}

func (f File) Destroy(ctx context.Context, hst host.Host, name Name) error {
	return fmt.Errorf("TODO File.Destroy")
}

func init() {
	IndividuallyManageableResourceTypeMap["File"] = File{}
}
