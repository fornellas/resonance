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

func (f *FileState) Load(ctx context.Context, host hostPkg.Host) error {

	return nil
}

func (f *FileState) Apply(ctx context.Context, host hostPkg.Host) error {
	return nil
}

type FileSingle struct{}

func (f *FileSingle) Load(ctx context.Context, host hostPkg.Host, name Name) (State, error) {
	fileState := &FileState{
		Path: string(name),
	}
	if err := fileState.Load(ctx, host); err != nil {
		return nil, err
	}
	return fileState, nil
}

func (f *FileSingle) Apply(ctx context.Context, host hostPkg.Host, state State) error {
	fileState := state.(*FileState)
	return fileState.Apply(ctx, host)
}

////

type SingleState interface {
	State
	Load(ctx context.Context, host hostPkg.Host) error
	Apply(ctx context.Context, host hostPkg.Host) error
}

type GenericSingle struct {
	getStateFn func(name Name) SingleState
}

func NewGenericSingle(getStateFn func(name Name) SingleState) *GenericSingle {
	return &GenericSingle{
		getStateFn: getStateFn,
	}
}

func (f *GenericSingle) Load(ctx context.Context, host hostPkg.Host, name Name) (State, error) {
	state := f.getStateFn(name)
	if err := state.Load(ctx, host); err != nil {
		return nil, err
	}
	return state, nil
}

func (f *GenericSingle) Apply(ctx context.Context, host hostPkg.Host, state State) error {
	singleState := state.(SingleState)
	return singleState.Apply(ctx, host)
}

// /
var FileType = NewSingleType("File", NewGenericSingle(func(name Name) SingleState {
	return &FileState{
		Path: string(name),
	}
}))

// var FileType = NewSingleType("File", &FileSingle{})
