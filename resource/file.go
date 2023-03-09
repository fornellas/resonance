package resource

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

// FileStateParameters for File
type FileStateParameters struct {
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

func (fds FileStateParameters) Validate() error {
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

func (fds FileStateParameters) GetUid(ctx context.Context, hst host.Host) (uint32, error) {
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

func (fds FileStateParameters) GetGid(ctx context.Context, hst host.Host) (uint32, error) {
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

// FileInternalState is InternalState for File
type FileInternalState struct {
	// Whether the file exists or not
	Exists bool
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

func (f File) GetFullState(ctx context.Context, hst host.Host, name Name) (FullState, error) {
	logger := log.GetLogger(ctx)

	path := string(name)

	stateParameters := FileStateParameters{}
	internalState := FileInternalState{}
	fullState := FullState{
		StateParameters: stateParameters,
		InternalState:   internalState,
	}

	// Content
	content, err := hst.ReadFile(ctx, path)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Debug("File not found")
			return fullState, nil
		}
		return FullState{}, err
	}
	stateParameters.Content = string(content)

	// FileInfo
	fileInfo, err := hst.Lstat(ctx, path)
	if err != nil {
		return FullState{}, err
	}
	stat_t := fileInfo.Sys().(*syscall.Stat_t)

	// Perm
	stateParameters.Perm = fileInfo.Mode()

	// Uid
	stateParameters.Uid = stat_t.Uid

	// Gid
	stateParameters.Gid = stat_t.Gid

	return fullState, nil
}

func (f File) DiffStates(
	desiredStateParameters StateParameters, currentFullState FullState,
) []diffmatchpatch.Diff {

	// // Path Hash
	// pathtHash := md5.New()
	// n, err := pathtHash.Write(content)
	// if err != nil {
	// 	return false, err
	// }
	// if n != len(content) {
	// 	return false, fmt.Errorf("unexpected write length when generating md5: expected %d, got %d", len(content), n)
	// }

	// // Instance Hash
	// fileStateHash := md5.New()
	// n, err := fileStateHash.Write([]byte(fileState.Content))
	// if err != nil {
	// 	return false, err
	// }
	// if n != len(fileState.Content) {
	// 	return false, fmt.Errorf("unexpected write length when generating md5: expected %d, got %d", len(fileState.Content), n)
	// }

	// // Compare Hash
	// if fmt.Sprintf("%v", pathtHash.Sum(nil)) != fmt.Sprintf("%v", fileStateHash.Sum(nil)) {
	// 	logger.Debug("Content differs")
	// 	checkResult = false
	// }

	panic(errors.New("File.DiffStates"))
}

func (f File) Apply(
	ctx context.Context, hst host.Host, name Name, stateParameters StateParameters,
) error {
	return errors.New("File.Apply")
}
func (f File) Destroy(ctx context.Context, hst host.Host, name Name) error {
	return errors.New("File.Destroy")
}

func init() {
	IndividuallyManageableResourceTypeMap["File"] = File{}
	ManageableResourcesStateParametersMap["File"] = FileStateParameters{}
	ManageableResourcesInternalStateMap["File"] = FileInternalState{}
}

// func (f File) Apply(ctx context.Context, hst host.Host, name Name, state State) error {
// 	nestedCtx := log.IndentLogger(ctx)
// 	path := string(name)

// 	// FileStateParameters
// 	fileState := state.(*FileStateParameters)

// 	// Content
// 	if err := hst.WriteFile(nestedCtx, path, []byte(fileState.Content), fileState.Perm); err != nil {
// 		return err
// 	}

// 	// Perm
// 	if err := hst.Chmod(nestedCtx, path, fileState.Perm); err != nil {
// 		return err
// 	}

// 	// FileInfo
// 	fileInfo, err := hst.Lstat(ctx, path)
// 	if err != nil {
// 		return err
// 	}
// 	stat_t := fileInfo.Sys().(*syscall.Stat_t)

// 	// Uid / Gid
// 	uid, err := fileState.GetUid(ctx, hst)
// 	if err != nil {
// 		return err
// 	}
// 	gid, err := fileState.GetGid(ctx, hst)
// 	if err != nil {
// 		return err
// 	}
// 	if stat_t.Uid != uid || stat_t.Gid != gid {
// 		if err := hst.Chown(ctx, path, int(fileState.Uid), int(fileState.Gid)); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// func (f File) Destroy(ctx context.Context, hst host.Host, name Name) error {
// 	nestedCtx := log.IndentLogger(ctx)
// 	path := string(name)
// 	err := hst.Remove(nestedCtx, path)
// 	if os.IsNotExist(err) {
// 		return nil
// 	}
// 	return err
// }
