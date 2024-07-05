package resources

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resources"
)

type IndividualState struct {
	Value string
}

func (is IndividualState) ValidateAndUpdate(ctx context.Context, hst host.Host) (resources.State, error) {
	return is, nil
}

type IndividualFuncValidateName struct {
	Name        resources.Name
	ReturnError error
}

func (ifvn IndividualFuncValidateName) String() string {
	return fmt.Sprintf("(%#v) %#v", ifvn.Name, ifvn.ReturnError)
}

type IndividualFuncGetState struct {
	Name        resources.Name
	ReturnState resources.State
	ReturnError error
}

func (ifgs IndividualFuncGetState) String() string {
	return fmt.Sprintf("(%#v) (%#v, %#v)", ifgs.Name, ifgs.ReturnState, ifgs.ReturnError)
}

type IndividualFuncConfigure struct {
	Name        resources.Name
	State       resources.State
	ReturnError error
}

func (ifc IndividualFuncConfigure) String() string {
	return fmt.Sprintf("(%#v, %#v) (%#v)", ifc.Name, ifc.State, ifc.ReturnError)
}

type IndividualFuncCall struct {
	ValidateName *IndividualFuncValidateName
	GetState     *IndividualFuncGetState
	Configure    *IndividualFuncConfigure
}

func (ifc IndividualFuncCall) String() string {
	ifcValue := reflect.ValueOf(ifc)
	ifcType := ifcValue.Type()
	for i := 0; i < ifcType.NumField(); i++ {
		field := ifcType.Field(i)
		name := field.Name
		value := ifcValue.Field(i)
		if !value.IsNil() {
			return fmt.Sprintf("%s: %v\n", name, value.Interface())
		}
	}
	panic("empty IndividualFuncCall")
}

type IndividualFuncCalls []IndividualFuncCall

func (ifcs IndividualFuncCalls) String() string {
	var s strings.Builder
	for i, ifc := range ifcs {
		fmt.Fprintf(&s, "%d: %v", i, ifc)
	}
	return s.String()
}

var IndividualT *testing.T
var IndividualExpectedFuncCalls IndividualFuncCalls

type Individual struct{}

func (i *Individual) getFuncCall() *IndividualFuncCall {
	if len(IndividualExpectedFuncCalls) == 0 {
		return nil
	}
	testFuncCall, expectedFuncCalls := IndividualExpectedFuncCalls[0], IndividualExpectedFuncCalls[1:]
	IndividualExpectedFuncCalls = expectedFuncCalls
	return &testFuncCall
}

func (i Individual) ValidateName(name resources.Name) error {
	funcCall := i.getFuncCall()
	if funcCall == nil {
		IndividualT.Fatalf("no more calls expected, got ValidateName(%#v)", name)
	}
	if funcCall.ValidateName == nil {
		IndividualT.Fatalf("unexpected call: got ValidateName(%#v), expected %#v", name, funcCall)
	}
	if funcCall.ValidateName.Name != name {
		IndividualT.Fatalf(
			"unexpected arguments: got ValidateName(%#v), expected ValidateName(%#v)",
			name, funcCall.ValidateName.Name,
		)
	}
	return funcCall.ValidateName.ReturnError
}

func (i Individual) GetState(ctx context.Context, hst host.Host, name resources.Name) (resources.State, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Test.GetState(%#v)", name)
	funcCall := i.getFuncCall()
	if funcCall == nil {
		IndividualT.Fatalf("no more calls expected, got GetState(%#v)", name)
	}
	if funcCall.GetState == nil {
		IndividualT.Fatalf("unexpected call: got GetState(%#v), expected %#v", name, funcCall)
	}
	if funcCall.GetState.Name != name {
		IndividualT.Fatalf(
			"unexpected arguments: got GetState(%#v), expected GetState(%#v)",
			name, funcCall.GetState.Name,
		)
	}
	return funcCall.GetState.ReturnState, funcCall.GetState.ReturnError
}

func (i Individual) Configure(
	ctx context.Context, hst host.Host, name resources.Name, state resources.State,
) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Test.Configure(%#v, %#v)", name, state)
	funcCall := i.getFuncCall()
	if funcCall == nil {
		IndividualT.Fatalf("no more calls expected, got Configure(%#v, %#v)", name, state)
	}
	if funcCall.Configure == nil {
		IndividualT.Fatalf("unexpected call: got Configure(%#v, %#v), expected %#v", name, state, funcCall)
	}
	if funcCall.Configure.Name != name || !reflect.DeepEqual(funcCall.Configure.State, state) {
		IndividualT.Fatalf(
			"unexpected arguments: got Configure(%#v, %#v), expected Configure(%#v, %#v)",
			name, state, funcCall.Configure.Name, funcCall.Configure.State,
		)
	}
	return funcCall.Configure.ReturnError
}

func SetupIndividualTypeMock(t *testing.T, individualFuncCalls []IndividualFuncCall) {
	IndividualT = t
	IndividualExpectedFuncCalls = individualFuncCalls
	t.Cleanup(func() {
		if len(IndividualExpectedFuncCalls) > 0 {
			t.Errorf("expected calls pending:\n%v", IndividualExpectedFuncCalls)
		}
	})
}

func init() {
	resources.IndividualResourceTypeMap["Individual"] = Individual{}
	resources.ResourcesStateMap["Individual"] = IndividualState{}
}
