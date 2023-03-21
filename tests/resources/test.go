package resources

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"
)

type TestState struct {
	Value string
}

func (ts TestState) ValidateAndUpdate(ctx context.Context, hst host.Host) (resource.State, error) {
	return ts, nil
}

type TestFuncValidateName struct {
	Name        resource.Name
	ReturnError error
}

func (tfvn TestFuncValidateName) String() string {
	return fmt.Sprintf("(%#v) %#v", tfvn.Name, tfvn.ReturnError)
}

type TestFuncGetState struct {
	Name        resource.Name
	ReturnState resource.State
	ReturnError error
}

func (tfgs TestFuncGetState) String() string {
	return fmt.Sprintf("(%#v) (%#v, %#v)", tfgs.Name, tfgs.ReturnState, tfgs.ReturnError)
}

type TestFuncConfigure struct {
	Name        resource.Name
	State       resource.State
	ReturnError error
}

func (tfa TestFuncConfigure) String() string {
	return fmt.Sprintf("(%#v, %#v) (%#v)", tfa.Name, tfa.State, tfa.ReturnError)
}

type TestFuncDestroy struct {
	Name        resource.Name
	ReturnError error
}

func (tfd TestFuncDestroy) String() string {
	return fmt.Sprintf("(%#v) (%#v)", tfd.Name, tfd.ReturnError)
}

type TestFuncCall struct {
	ValidateName *TestFuncValidateName
	GetState     *TestFuncGetState
	Configure    *TestFuncConfigure
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

func (t Test) ValidateName(name resource.Name) error {
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

func (t Test) GetState(ctx context.Context, hst host.Host, name resource.Name) (resource.State, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Test.GetState(%#v)", name)
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

func (t Test) Configure(
	ctx context.Context, hst host.Host, name resource.Name, state resource.State,
) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Test.Configure(%#v, %#v)", name, state)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		TestT.Fatalf("no more calls expected, got Configure(%#v, %#v)", name, state)
	}
	if funcCall.Configure == nil {
		TestT.Fatalf("unexpected call: got Configure(%#v, %#v), expected %#v", name, state, funcCall)
	}
	if funcCall.Configure.Name != name || !reflect.DeepEqual(funcCall.Configure.State, state) {
		TestT.Fatalf(
			"unexpected arguments: got Configure(%#v, %#v), expected Configure(%#v, %#v)",
			name, state, funcCall.Configure.Name, funcCall.Configure.State,
		)
	}
	return funcCall.Configure.ReturnError
}

func (t Test) Destroy(ctx context.Context, hst host.Host, name resource.Name) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Test.Destroy(%#v)", name)
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
	resource.IndividuallyManageableResourceTypeMap["Test"] = Test{}
	resource.ManageableResourcesStateMap["Test"] = TestState{}
}
