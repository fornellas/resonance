package resonance

import (
	"context"

	"errors"

	"github.com/fornellas/resonance/host"
)

// Generic Interfaces

type ResourceState[T any] interface {
	Validate() error
	Satisfies(resource ResourceState[T]) bool
}

type SingleResource[T any] interface {
	ResourceState[T]
	Load(context.Context, host.Host) error
	Resolve(context.Context, host.Host) error
	Apply(context.Context, host.Host) error
}

type GroupResource[T ResourceState[T]] interface {
	Load(context.Context, host.Host, []T) error
	Resolve(context.Context, host.Host, []T) error
	Apply(context.Context, host.Host, []T) error
}

// File

type FileState struct {
}

func (f *FileState) Validate() error {
	panic(errors.New("TODO"))
}

func (f *FileState) Satisfies(otherFileState *FileState) bool {
	panic(errors.New("TODO"))
}

// Resaurce Schema

type ResourceSchema struct {
	File FileState
	// APTPackage APTPackage
}

type ResourcesSchema []ResourceSchema

// Blueprint

type Step struct {
}
