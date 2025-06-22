package resonance

import (
	"context"
	"fmt"

	"github.com/fornellas/resonance/host/types"
)

////////////////////////////////////////////////////////////////////////////////////////////////////
// State interfaces
////////////////////////////////////////////////////////////////////////////////////////////////////

type State interface {
	Validate() error
}

type StateSatisfier[S State] interface {
	State
	Satisfies(StateSatisfier[S]) bool
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// State
////////////////////////////////////////////////////////////////////////////////////////////////////

// File State

type FileState struct {
	Path string
}

func (f *FileState) Validate() error { return nil }

// APT Package State

type AptPackageState struct {
	Path string
}

func (f *AptPackageState) Validate() error { return nil }

func (f *AptPackageState) Satisfies(AptPackageState) bool { return false }

////////////////////////////////////////////////////////////////////////////////////////////////////
// Schema
////////////////////////////////////////////////////////////////////////////////////////////////////

type StateDefinition struct {
	File       *FileState
	AptPackage *AptPackageState
}

func (s *StateDefinition) State() (State, error) {
	states := []State{}
	if s.File != nil {
		states = append(states, s.File)
	}
	if s.AptPackage != nil {
		states = append(states, s.AptPackage)
	}
	switch len(states) {
	case 0:
		return nil, fmt.Errorf("no state defined")
	case 1:
		return states[0], nil
	default:
		return nil, fmt.Errorf("only one state can be defined")
	}
}

type StatesDefinition []StateDefinition

func LoadStatesDefinition(path string) (StatesDefinition, error) {
	// TODO load from yaml files at path
	// TODO ensure unique name for each state type
	return StatesDefinition{}, nil
}

func (s StatesDefinition) States() ([]State, error) {
	states := make([]State, len(s))
	for i, stateSchema := range s {
		var err error
		states[i], err = stateSchema.State()
		if err != nil {
			return nil, err
		}
	}
	return states, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Provisioner interfaces
////////////////////////////////////////////////////////////////////////////////////////////////////

type SingleProvisioner interface {
	Load(context.Context, State) error
	Resolve(context.Context, State) error
	Apply(context.Context, State) error
}

type GroupProvisioner interface {
	Load(context.Context, []State) error
	Resolve(context.Context, []State) error
	Apply(context.Context, []State) error
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Provisioners
////////////////////////////////////////////////////////////////////////////////////////////////////

// File Provisioner

type FileProvisioner struct {
}

func NewFileProvisioner(types.Host) (*FileProvisioner, error) {
	return &FileProvisioner{}, nil
}

func (f *FileProvisioner) Load(context.Context, FileState) error    { return nil }
func (f *FileProvisioner) Resolve(context.Context, FileState) error { return nil }
func (f *FileProvisioner) Apply(context.Context, FileState) error   { return nil }

// APT Package Provisioner

type AptPackageProvisioner struct {
}

func NewAptPackageProvisioner(types.Host) (*AptPackageProvisioner, error) {
	return &AptPackageProvisioner{}, nil
}

func (f *AptPackageProvisioner) Load(context.Context, []AptPackageState) error    { return nil }
func (f *AptPackageProvisioner) Resolve(context.Context, []AptPackageState) error { return nil }
func (f *AptPackageProvisioner) Apply(context.Context, []AptPackageState) error   { return nil }

////////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////////

type Action interface {
	Apply(context.Context)
}

type SingleAction struct {
	State             State
	SingleProvisioner SingleProvisioner
}

func (a *SingleAction) Apply(context.Context) error { return nil }

type GroupAction struct {
	State            State
	GroupProvisioner GroupProvisioner
}

func (a *GroupAction) Apply(context.Context) error { return nil }

type Plan []Action
