package resource

import (
	"context"
	"fmt"
	"os"

	"github.com/fornellas/resonance/host"
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

// FileState for File
type FileState struct {
	Md5   []byte      `yaml:"md5"`
	Perm  os.FileMode `yaml:"perm"`
	Uid   uint32      `yaml:"uid"`
	User  string      `yaml:"user"`
	Gid   uint32      `yaml:"gid"`
	Group string      `yaml:"group"`
}

// File resource manages files.
type File struct{}

func (f File) MergeApply() bool {
	return false
}

// func (f File) GetDesiredState(parameters yaml.Node) (State, error) {
// 	var fileParams FileParams
// 	fileState := FileState{}

// 	if err := parameters.Decode(&fileParams); err != nil {
// 		return fileState, err
// 	}

// 	h := md5.New()
// 	n, err := h.Write([]byte(fileParams.Content))
// 	if err != nil {
// 		return fileState, err
// 	}
// 	if n != len(fileParams.Content) {
// 		return fileState, fmt.Errorf("unexpected write length when generating md5: expected %d, got %d", len(fileParams.Content), n)
// 	}
// 	fileState.Md5 = h.Sum(nil)

// 	if fileParams.Perm == 0 {
// 		return fileState, fmt.Errorf("perm must be set")
// 	}
// 	fileState.Perm = fileParams.Perm

// 	if fileParams.Uid != 0 && fileParams.User != "" {
// 		return fileState, fmt.Errorf("can't set both uid and user")
// 	}
// 	if fileParams.Uid == 0 && fileParams.User == "" {
// 		return fileState, fmt.Errorf("must set eithre uid or user")
// 	}
// 	fileState.Uid = fileParams.Uid
// 	fileState.User = fileParams.User

// 	if fileParams.Gid != 0 && fileParams.Group != "" {
// 		return fileState, fmt.Errorf("can't set both gid and group")
// 	}
// 	if fileParams.Gid == 0 && fileParams.Group == "" {
// 		return fileState, fmt.Errorf("must set eithre gid or group")
// 	}
// 	fileState.Gid = fileParams.Gid
// 	fileState.Group = fileParams.Group

// 	return fileState, nil
// }

// func (f File) GetState(ctx context.Context, hst host.Host, name Name) (State, error) {
// 	fileState := FileState{}

// 	path := name.String()

// 	// Md5
// 	h := md5.New()
// 	content, err := hst.ReadFile(ctx, path)
// 	if err != nil {
// 		if !errors.Is(err, fs.ErrNotExist) {
// 			return fileState, err
// 		}
// 		return fileState, nil
// 	} else {
// 		n, err := h.Write(content)
// 		if err != nil {
// 			return fileState, err
// 		}
// 		if n != len(content) {
// 			return fileState, fmt.Errorf("unexpected write length when generating md5: expected %d, got %d", len(content), n)
// 		}
// 	}
// 	fileState.Md5 = h.Sum(nil)

// 	// Perm
// 	fileInfo, err := hst.Lstat(ctx, path)
// 	if err != nil {
// 		return fileState, err
// 	}
// 	fileState.Perm = fileInfo.Mode()

// 	// Uid
// 	fileState.Uid = fileInfo.Sys().(*syscall.Stat_t).Uid

// 	// User
// 	u, err := user.LookupId(strconv.Itoa(int(fileState.Uid)))
// 	if err != nil {
// 		return fileState, err
// 	}
// 	fileState.User = u.Username

// 	// Gid
// 	fileState.Gid = fileInfo.Sys().(*syscall.Stat_t).Gid

// 	// Group
// 	g, err := user.LookupGroupId(strconv.Itoa(int(fileState.Gid)))
// 	if err != nil {
// 		return fileState, err
// 	}
// 	fileState.Group = g.Name

// 	return fileState, nil
// }

func (f File) Check(ctx context.Context, hst host.Host, instance Instance) (bool, error) {
	return false, fmt.Errorf("TODO File.Check")
}

func (f File) Apply(ctx context.Context, hst host.Host, instances []Instance) error {
	// TODO use Host interface
	// fileParams := parameters.(FileParams)

	// if err := os.WriteFile(name, fileParams.Content, fileParams.Perm); err != nil {
	// 	return err
	// }
	// return nil
	return fmt.Errorf("TODO File.Apply")
}

func (f File) Refresh(ctx context.Context, hst host.Host, name Name) error {
	return nil
}

func (f File) Destroy(ctx context.Context, hst host.Host, name Name) error {
	return fmt.Errorf("TODO File.Destroy")
}

func init() {
	TypeToManageableResource["File"] = File{}
}
