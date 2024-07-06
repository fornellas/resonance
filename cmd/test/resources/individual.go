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

type IndividualFnValidateName struct {
	Name        resources.Name
	ReturnError error
}

func (ifvn IndividualFnValidateName) String() string {
	return fmt.Sprintf("(%#v) %#v", ifvn.Name, ifvn.ReturnError)
}

type IndividualFnGetState struct {
	Name        resources.Name
	ReturnState resources.State
	ReturnError error
}

func (ifgs IndividualFnGetState) String() string {
	return fmt.Sprintf("(%#v) (%#v, %#v)", ifgs.Name, ifgs.ReturnState, ifgs.ReturnError)
}

type IndividualFnConfigure struct {
	Name        resources.Name
	State       resources.State
	ReturnError error
}

func (ifc IndividualFnConfigure) String() string {
	return fmt.Sprintf("(%#v, %#v) (%#v)", ifc.Name, ifc.State, ifc.ReturnError)
}

type IndividualFnCall struct {
	ValidateName *IndividualFnValidateName
	GetState     *IndividualFnGetState
	Configure    *IndividualFnConfigure
}

func (ifc IndividualFnCall) String() string {
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
	panic("empty IndividualFnCall")
}

type IndividualFnCalls []IndividualFnCall

func (ifcs IndividualFnCalls) String() string {
	var s strings.Builder
	for i, ifc := range ifcs {
		fmt.Fprintf(&s, "%d: %v", i, ifc)
	}
	return s.String()
}

var IndividualT *testing.T
var IndividualExpectedFnCalls IndividualFnCalls

type Individual struct{}

func (i *Individual) getFnCall() *IndividualFnCall {
	if len(IndividualExpectedFnCalls) == 0 {
		return nil
	}
	testFnCall, expectedFnCalls := IndividualExpectedFnCalls[0], IndividualExpectedFnCalls[1:]
	IndividualExpectedFnCalls = expectedFnCalls
	return &testFnCall
}

func (i Individual) ValidateName(name resources.Name) error {
	funcCall := i.getFnCall()
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
	logger := log.MustLoggerIndented(ctx)
	logger.Debug("Individual.GetState", "name", name)
	funcCall := i.getFnCall()
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
	logger := log.MustLoggerIndented(ctx)
	logger.Debug("Individual.Configure", "name", name, "state", state)
	funcCall := i.getFnCall()
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

func SetupIndividualTypeMock(t *testing.T, individualFnCalls []IndividualFnCall) {
	IndividualT = t
	IndividualExpectedFnCalls = individualFnCalls
	t.Cleanup(func() {
		if len(IndividualExpectedFnCalls) > 0 {
			t.Errorf("expected calls pending:\n%v", IndividualExpectedFnCalls)
		}
	})
}

func init() {
	resources.IndividualResourceTypeMap["Individual"] = Individual{}
	resources.ResourcesStateMap["Individual"] = IndividualState{}
}
