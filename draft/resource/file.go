package resource

import (
	"context"

	hostPkg "github.com/fornellas/resonance/draft/host"
)

type FileState struct {
	Path string `json:"path"`
}

func (f *FileState) Id() *Id {
	return &Id{
		Type: FileType,
		Name: Name(f.Path),
	}
}

type FileSingle struct{}

func (f *FileSingle) Load(ctx context.Context, host hostPkg.Host, name Name) (State, error) {
	fileState := &FileState{
		Path: string(name),
	}
	// TODO
	return fileState, nil
}

func (f *FileSingle) Apply(ctx context.Context, host hostPkg.Host, state State) error {
	// fileState := state.(*FileState)
	// TODO
	return nil
}

var FileType = NewSingleType("File", &FileSingle{})
