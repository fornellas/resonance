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

type IndividualState struct {
	Value string
}

func (is IndividualState) ValidateAndUpdate(ctx context.Context, hst host.Host) (resource.State, error) {
	return is, nil
}

type IndividualFuncValidateName struct {
	Name        resource.Name
	ReturnError error
}

func (ifvn IndividualFuncValidateName) String() string {
	return fmt.Sprintf("(%#v) %#v", ifvn.Name, ifvn.ReturnError)
}

type IndividualFuncGetState struct {
	Name        resource.Name
	ReturnState resource.State
	ReturnError error
}

func (ifgs IndividualFuncGetState) String() string {
	return fmt.Sprintf("(%#v) (%#v, %#v)", ifgs.Name, ifgs.ReturnState, ifgs.ReturnError)
}

type IndividualFuncConfigure struct {
	Name        resource.Name
	State       resource.State
	ReturnError error
}

func (ifc IndividualFuncConfigure) String() string {
	return fmt.Sprintf("(%#v, %#v) (%#v)", ifc.Name, ifc.State, ifc.ReturnError)
}

type IndividualFuncDestroy struct {
	Name        resource.Name
	ReturnError error
}

func (ifd IndividualFuncDestroy) String() string {
	return fmt.Sprintf("(%#v) (%#v)", ifd.Name, ifd.ReturnError)
}

type IndividualFuncCall struct {
	ValidateName *IndividualFuncValidateName
	GetState     *IndividualFuncGetState
	Configure    *IndividualFuncConfigure
	Destroy      *IndividualFuncDestroy
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

func (i Individual) ValidateName(name resource.Name) error {
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

func (i Individual) GetState(ctx context.Context, hst host.Host, name resource.Name) (resource.State, error) {
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
	ctx context.Context, hst host.Host, name resource.Name, state resource.State,
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

func (i Individual) Destroy(ctx context.Context, hst host.Host, name resource.Name) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Test.Destroy(%#v)", name)
	funcCall := i.getFuncCall()
	if funcCall == nil {
		IndividualT.Fatalf("no more calls expected, got Destroy(%#v)", name)
	}
	if funcCall.Destroy == nil {
		IndividualT.Fatalf("unexpected call: got Destroy(%#v), expected %#v", name, funcCall)
	}
	if funcCall.Destroy.Name != name {
		IndividualT.Fatalf(
			"unexpected arguments: got Destroy(%#v), expected Destroy(%#v)",
			name, funcCall.Destroy.Name,
		)
	}
	return funcCall.Destroy.ReturnError
}

func SetupIndividualType(t *testing.T, individualFuncCalls []IndividualFuncCall) {
	IndividualT = t
	IndividualExpectedFuncCalls = individualFuncCalls
	t.Cleanup(func() {
		if len(IndividualExpectedFuncCalls) > 0 {
			t.Errorf("expected calls pending:\n%v", IndividualExpectedFuncCalls)
		}
	})
}

func init() {
	resource.IndividuallyManageableResourceTypeMap["Individual"] = Individual{}
	resource.ManageableResourcesStateMap["Individual"] = IndividualState{}
}
