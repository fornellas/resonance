package resource

import (
	"context"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/fornellas/resonance/host"
)

// TestState for Test
type TestState struct {
}

func (ts TestState) Validate() error {
	return nil
}

type TestFuncValidateName struct {
	Name        Name
	ReturnError error
}

type TestFuncGetState struct {
	Name        Name
	ReturnState State
	ReturnError error
}

type TestFuncDiffStates struct {
	DesiredState State
	CurrentState State
	ReturnDiffs  []diffmatchpatch.Diff
	ReturnError  error
}

type TestFuncApply struct {
	Name        Name
	State       State
	ReturnError error
}

type TestFuncDestroy struct {
	Name        Name
	ReturnError error
}

type TestFuncCall struct {
	ValidateName *TestFuncValidateName
	GetState     *TestFuncGetState
	DiffStates   *TestFuncDiffStates
	Apply        *TestFuncApply
	Destroy      *TestFuncDestroy
}

// Test resource.
type Test struct {
	T                 *testing.T
	ExpectedFuncCalls []TestFuncCall
}

var TestInstance Test

func (t Test) ValidateName(name Name) error {
	return nil
}

func (t Test) GetState(ctx context.Context, hst host.Host, name Name) (State, error) {
	return &TestState{}, nil
}

func (t Test) DiffStates(
	ctx context.Context, hst host.Host,
	desiredState State, currentState State,
) ([]diffmatchpatch.Diff, error) {
	return []diffmatchpatch.Diff{}, nil
}

func (t Test) Apply(
	ctx context.Context, hst host.Host, name Name, state State,
) error {
	return nil
}

func (t Test) Destroy(ctx context.Context, hst host.Host, name Name) error {
	return nil
}

func init() {
	IndividuallyManageableResourceTypeMap["Test"] = TestInstance
	ManageableResourcesStateMap["Test"] = TestState{}
}
