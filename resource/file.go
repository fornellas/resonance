package resource

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"strconv"
	"syscall"

	"github.com/fornellas/resonance/host"
)

// FileParams for File
type FileParams struct {
	// Contents of the file
	Content []byte `yaml:"content"`
	// File permissions
	Perm os.FileMode `yaml:"perm"`
	// User ID owner of the file
	Uid int `yaml:"uid"`
	// User name owner of the file
	User string `yaml:"user"`
	// Group ID owner of the file
	Gid int `yaml:"gid"`
	// Group name owner of the file
	Group string `yaml:"group"`
}

// FileState for File
type FileState struct {
	Md5   []byte      `yaml:"md5"`
	Perm  os.FileMode `yaml:"perm"`
	Uid   os.FileMode `yaml:"uid"`
	User  string      `yaml:"user"`
	Gid   os.FileMode `yaml:"gid"`
	Group string      `yaml:"group"`
}

// File resource manages files.
type File struct{}

func (f File) AlwaysMergeApply() bool {
	return false
}

func (f File) ReadState(
	ctx context.Context,
	host host.Host,
	instance Instance,
) (State, error) {
	fileState := FileState{}

	path := instance.Name.String()

	// Md5
	h := md5.New()
	content, err := host.ReadFile(ctx, path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fileState, err
		}
		return fileState, nil
	} else {
		n, err := h.Write(content)
		if err != nil {
			return fileState, err
		}
		if n != len(content) {
			return fileState, fmt.Errorf("unexpected write length when generating md5: expected %d, got %d", len(content), n)
		}
	}
	fileState.Md5 = h.Sum(nil)

	// Perm
	fileInfo, err := host.Lstat(ctx, path)
	if err != nil {
		return fileState, err
	}
	fileState.Perm = fileInfo.Mode()

	// Uid
	fileState.Uid = os.FileMode(fileInfo.Sys().(*syscall.Stat_t).Uid)

	// User
	u, err := user.LookupId(strconv.Itoa(int(fileState.Uid)))
	if err != nil {
		return fileState, err
	}
	fileState.User = u.Username

	// Gid
	fileState.Gid = os.FileMode(fileInfo.Sys().(*syscall.Stat_t).Gid)

	// Group
	g, err := user.LookupGroupId(strconv.Itoa(int(fileState.Gid)))
	if err != nil {
		return fileState, err
	}
	fileState.Group = g.Name

	return fileState, nil
}

func (f File) Apply(
	ctx context.Context,
	host host.Host,
	instances []Instance,
) error {
	// TODO use Host interface
	// fileParams := parameters.(FileParams)

	// if err := os.WriteFile(name, fileParams.Content, fileParams.Perm); err != nil {
	// 	return err
	// }
	// return nil
	return fmt.Errorf("TODO File.Apply")
}

func (f File) Destroy(
	ctx context.Context,
	host host.Host,
	instances []Instance,
) error {
	return fmt.Errorf("TODO File.Destroy")
}
