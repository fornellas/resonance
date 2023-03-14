package resource

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/fornellas/resonance/host"
)

type TestState struct {
	Value string
}

func (ts TestState) Validate() error {
	return nil
}

type TestFuncValidateName struct {
	Name        Name
	ReturnError error
}

func (tfvn TestFuncValidateName) String() string {
	return fmt.Sprintf("(%#v) %#v", tfvn.Name, tfvn.ReturnError)
}

type TestFuncGetState struct {
	Name        Name
	ReturnState State
	ReturnError error
}

func (tfgs TestFuncGetState) String() string {
	return fmt.Sprintf("(%#v) (%#v, %#v)", tfgs.Name, tfgs.ReturnState, tfgs.ReturnError)
}

type TestFuncDiffStates struct {
	DesiredState State
	CurrentState State
	ReturnDiffs  []diffmatchpatch.Diff
	ReturnError  error
}

func (tfds TestFuncDiffStates) String() string {
	return fmt.Sprintf(
		"(%#v, %#v) (%#v, %#v)",
		tfds.DesiredState, tfds.CurrentState, tfds.ReturnDiffs, tfds.ReturnError,
	)
}

type TestFuncApply struct {
	Name        Name
	State       State
	ReturnError error
}

func (tfa TestFuncApply) String() string {
	return fmt.Sprintf("(%#v, %#v) (%#v)", tfa.Name, tfa.State, tfa.ReturnError)
}

type TestFuncDestroy struct {
	Name        Name
	ReturnError error
}

func (tfd TestFuncDestroy) String() string {
	return fmt.Sprintf("(%#v) (%#v)", tfd.Name, tfd.ReturnError)
}

type TestFuncCall struct {
	ValidateName *TestFuncValidateName
	GetState     *TestFuncGetState
	DiffStates   *TestFuncDiffStates
	Apply        *TestFuncApply
	Destroy      *TestFuncDestroy
}

func (tfc TestFuncCall) String() string {
	tfcValue := reflect.ValueOf(tfc)
	tfcType := tfcValue.Type()
	for i := 0; i < tfcType.NumField(); i++ {
		field := tfcType.Field(i)
		name := field.Name
		value := tfcValue.Field(i)
		if !value.IsNil() {
			return fmt.Sprintf("%s: %v\n", name, value.Interface())
		}
	}
	panic("empty TestFuncCall")
}

type TestFuncCalls []TestFuncCall

func (tfcs TestFuncCalls) String() string {
	var s strings.Builder
	for i, tfc := range tfcs {
		fmt.Fprintf(&s, "%d: %v", i, tfc)
	}
	return s.String()
}

var TestT *testing.T
var TestExpectedFuncCalls TestFuncCalls

type Test struct{}

func (t *Test) getFuncCall() *TestFuncCall {
	if len(TestExpectedFuncCalls) == 0 {
		return nil
	}
	testFuncCall, expectedFuncCalls := TestExpectedFuncCalls[0], TestExpectedFuncCalls[1:]
	TestExpectedFuncCalls = expectedFuncCalls
	return &testFuncCall
}

func (t Test) ValidateName(name Name) error {
	funcCall := t.getFuncCall()
	if funcCall == nil {
		TestT.Fatalf("no more calls expected, got ValidateName(%#v)", name)
	}
	if funcCall.ValidateName == nil {
		TestT.Fatalf("unexpected call: got ValidateName(%#v), expected %#v", name, funcCall)
	}
	if funcCall.ValidateName.Name != name {
		TestT.Fatalf(
			"unexpected arguments: got ValidateName(%#v), expected ValidateName(%#v)",
			name, funcCall.ValidateName.Name,
		)
	}
	return funcCall.ValidateName.ReturnError
}

func (t Test) GetState(ctx context.Context, hst host.Host, name Name) (State, error) {
	funcCall := t.getFuncCall()
	if funcCall == nil {
		TestT.Fatalf("no more calls expected, got GetState(%#v)", name)
	}
	if funcCall.GetState == nil {
		TestT.Fatalf("unexpected call: got GetState(%#v), expected %#v", name, funcCall)
	}
	if funcCall.GetState.Name != name {
		TestT.Fatalf(
			"unexpected arguments: got GetState(%#v), expected GetState(%#v)",
			name, funcCall.GetState.Name,
		)
	}
	return funcCall.GetState.ReturnState, funcCall.GetState.ReturnError
}

func (t Test) DiffStates(
	ctx context.Context, hst host.Host,
	desiredState State, currentState State,
) ([]diffmatchpatch.Diff, error) {
	funcCall := t.getFuncCall()
	if funcCall == nil {
		TestT.Fatalf("no more calls expected, got DiffStates(%#v, %#v)", desiredState, currentState)
	}
	if funcCall.DiffStates == nil {
		TestT.Fatalf("unexpected call: got DiffStates(%#v, %#v), expected %#v", desiredState, currentState, funcCall)
	}
	if !reflect.DeepEqual(funcCall.DiffStates.DesiredState, desiredState) || !reflect.DeepEqual(funcCall.DiffStates.CurrentState, currentState) {
		TestT.Fatalf(
			"unexpected arguments: got DiffStates(%#v, %#v), expected DiffStates(%#v, %#v)",
			desiredState, currentState, funcCall.DiffStates.DesiredState, funcCall.DiffStates.CurrentState,
		)
	}
	return funcCall.DiffStates.ReturnDiffs, funcCall.DiffStates.ReturnError
}

func (t Test) Apply(
	ctx context.Context, hst host.Host, name Name, state State,
) error {
	funcCall := t.getFuncCall()
	if funcCall == nil {
		TestT.Fatalf("no more calls expected, got Apply(%#v, %#v)", name, state)
	}
	if funcCall.Apply == nil {
		TestT.Fatalf("unexpected call: got Apply(%#v, %#v), expected %#v", name, state, funcCall)
	}
	if funcCall.Apply.Name != name || !reflect.DeepEqual(funcCall.Apply.State, state) {
		TestT.Fatalf(
			"unexpected arguments: got Apply(%#v, %#v), expected Apply(%#v, %#v)",
			name, state, funcCall.Apply.Name, funcCall.Apply.State,
		)
	}
	return funcCall.Apply.ReturnError
}

func (t Test) Destroy(ctx context.Context, hst host.Host, name Name) error {
	funcCall := t.getFuncCall()
	if funcCall == nil {
		TestT.Fatalf("no more calls expected, got Destroy(%#v)", name)
	}
	if funcCall.Destroy == nil {
		TestT.Fatalf("unexpected call: got Destroy(%#v), expected %#v", name, funcCall)
	}
	if funcCall.Destroy.Name != name {
		TestT.Fatalf(
			"unexpected arguments: got Destroy(%#v), expected Destroy(%#v)",
			name, funcCall.Destroy.Name,
		)
	}
	return funcCall.Destroy.ReturnError
}

func init() {
	IndividuallyManageableResourceTypeMap["Test"] = Test{}
	ManageableResourcesStateMap["Test"] = TestState{}
}
