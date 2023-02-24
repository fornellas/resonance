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
	Content []byte
	// File permissions
	Perm os.FileMode
	// User name owner of the file
	User string
	// User ID owner of the file
	Uid int
	// Group name owner of the file
	Group string
	// Group ID owner of the file
	Gid int
}

// FileState for File
type FileState struct {
	Md5 []byte
}

// File resource manages files.
type File struct {
	// Resource
}

func (f File) AlwaysMergeApply() bool {
	return false
}

func (f File) ReadState(
	ctx context.Context,
	host host.Host,
	instance Instance,
) (State, error) {
	// TODO use Host interface
	// fileParams := parameters.(FileParams)

	// f, err := os.Open(fileParams.Path)
	// if err != nil {
	// 	return nil, err
	// }
	// defer f.Close()

	// h := md5.New()
	// if _, err := io.Copy(h, f); err != nil {
	// 	return nil, err
	// }

	// return FileState{
	// 	Md5: h.Sum(nil),
	// }, nil
	return FileState{}, fmt.Errorf("TODO File.ReadState")
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
