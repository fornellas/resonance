package resource

import (
	"context"
	"fmt"
	"os"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/state"
)

type FileParams struct {
	Content []byte
	Perm    os.FileMode
	User    string
	Uid     int
	Group   string
	Gid     int
}

type FileState struct {
	Md5 []byte
}

type File struct {
	// Resource
}

func (c File) ReadState(
	ctx context.Context,
	host host.Host,
	name string,
	parameters []Parameter,
) (state.ResourceState, error) {
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

func (c File) Apply(
	ctx context.Context,
	host host.Host,
	name string,
	parameters []Parameter,
) error {
	// fileParams := parameters.(FileParams)

	// if err := os.WriteFile(name, fileParams.Content, fileParams.Perm); err != nil {
	// 	return err
	// }
	// return nil
	return fmt.Errorf("TODO File.Apply")
}
