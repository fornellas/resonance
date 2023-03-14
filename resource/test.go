package resource

import (
	"context"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/fornellas/resonance/host"
)

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

type Test struct {
	T                 *testing.T
	ExpectedFuncCalls []TestFuncCall
}

func (t *Test) FinalAssert() {
	if len(t.ExpectedFuncCalls) > 0 {
		t.T.Fatalf("expected calls pending: %v", t.ExpectedFuncCalls)
	}
}

func (t *Test) getFuncCall() TestFuncCall {
	if len(t.ExpectedFuncCalls) == 0 {
		t.T.Fatalf("No more calls expected")
	}
	testFuncCall, expectedFuncCalls := t.ExpectedFuncCalls[0], t.ExpectedFuncCalls[1:]
	t.ExpectedFuncCalls = expectedFuncCalls
	return testFuncCall
}

func (t Test) ValidateName(name Name) error {
	funcCall := t.getFuncCall()
	if funcCall.ValidateName.Name != name {
		t.T.Fatalf(
			"unexpected arguments: got ValidateName(%v), expected ValidateName(%v)",
			name, funcCall.ValidateName.Name,
		)
	}
	return funcCall.ValidateName.ReturnError
}

func (t Test) GetState(ctx context.Context, hst host.Host, name Name) (State, error) {
	panic("Test.GetState")
}

func (t Test) DiffStates(
	ctx context.Context, hst host.Host,
	desiredState State, currentState State,
) ([]diffmatchpatch.Diff, error) {
	panic("Test.DiffStates")
}

func (t Test) Apply(
	ctx context.Context, hst host.Host, name Name, state State,
) error {
	panic("Test.Apply")
}

func (t Test) Destroy(ctx context.Context, hst host.Host, name Name) error {
	panic("Test.Destroy")
}

var TestInstance Test

func init() {
	IndividuallyManageableResourceTypeMap["Test"] = TestInstance
	ManageableResourcesStateMap["Test"] = TestState{}
}
