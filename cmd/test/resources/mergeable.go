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

type MergeableState struct {
	Value string
}

func (ms MergeableState) ValidateAndUpdate(ctx context.Context, hst host.Host) (resources.State, error) {
	return ms, nil
}

type MergeableFnValidateName struct {
	Name        resources.Name
	ReturnError error
}

func (mfvn MergeableFnValidateName) String() string {
	return fmt.Sprintf("(%#v) %#v", mfvn.Name, mfvn.ReturnError)
}

type MergeableFnGetStates struct {
	Names              resources.Names
	ReturnNameStateMap map[resources.Name]resources.State
	ReturnError        error
}

func (mfgs MergeableFnGetStates) String() string {
	return fmt.Sprintf("(%#v) (%#v, %#v)", mfgs.Names, mfgs.ReturnNameStateMap, mfgs.ReturnError)
}

type MergeableFnConfigureAll struct {
	ActionNameStateMap map[resources.Action]map[resources.Name]resources.State
	ReturnError        error
}

func (mfca MergeableFnConfigureAll) String() string {
	return fmt.Sprintf("(%#v) (%#v)", mfca.ActionNameStateMap, mfca.ReturnError)
}

type MergeableFnDestroy struct {
	Name        resources.Name
	ReturnError error
}

func (ifd MergeableFnDestroy) String() string {
	return fmt.Sprintf("(%#v) (%#v)", ifd.Name, ifd.ReturnError)
}

type MergeableFnCall struct {
	ValidateName *MergeableFnValidateName
	GetStates    *MergeableFnGetStates
	ConfigureAll *MergeableFnConfigureAll
}

func (mfc MergeableFnCall) String() string {
	mfcValue := reflect.ValueOf(mfc)
	mfcType := mfcValue.Type()
	for i := 0; i < mfcType.NumField(); i++ {
		field := mfcType.Field(i)
		name := field.Name
		value := mfcValue.Field(i)
		if !value.IsNil() {
			return fmt.Sprintf("%s: %v\n", name, value.Interface())
		}
	}
	panic("empty MergeableFnCall")
}

type MergeableFnCalls []MergeableFnCall

func (mfcs MergeableFnCalls) String() string {
	var s strings.Builder
	for i, mfc := range mfcs {
		fmt.Fprintf(&s, "%d: %v", i, mfc)
	}
	return s.String()
}

var MergeableT *testing.T
var MergeableExpectedFnCalls MergeableFnCalls

type Mergeable struct{}

func (m *Mergeable) getFnCall() *MergeableFnCall {
	if len(MergeableExpectedFnCalls) == 0 {
		return nil
	}
	testFnCall, expectedFnCalls := MergeableExpectedFnCalls[0], MergeableExpectedFnCalls[1:]
	MergeableExpectedFnCalls = expectedFnCalls
	return &testFnCall
}

func (m Mergeable) ValidateName(name resources.Name) error {
	funcCall := m.getFnCall()
	if funcCall == nil {
		MergeableT.Fatalf("no more calls expected, got ValidateName(%#v)", name)
	}
	if funcCall.ValidateName == nil {
		MergeableT.Fatalf("unexpected call: got ValidateName(%#v), expected %#v", name, funcCall)
	}
	if funcCall.ValidateName.Name != name {
		MergeableT.Fatalf(
			"unexpected arguments: got ValidateName(%#v), expected ValidateName(%#v)",
			name, funcCall.ValidateName.Name,
		)
	}
	return funcCall.ValidateName.ReturnError
}

func (m Mergeable) GetStates(
	ctx context.Context, hst host.Host,
	names resources.Names,
) (map[resources.Name]resources.State, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Mergeable.GetStates", "names", names)
	funcCall := m.getFnCall()
	if funcCall == nil {
		MergeableT.Fatalf("no more calls expected, got GetStates(%#v)", names)
	}
	if funcCall.GetStates == nil {
		MergeableT.Fatalf("unexpected call: got GetStates(%#v), expected %#v", names, funcCall)
	}
	if !reflect.DeepEqual(funcCall.GetStates.Names, names) {
		MergeableT.Fatalf(
			"unexpected arguments: got GetStates(%#v), expected GetStates(%#v)",
			names, funcCall.GetStates.Names,
		)
	}
	return funcCall.GetStates.ReturnNameStateMap, funcCall.GetStates.ReturnError
}

func (m Mergeable) ConfigureAll(
	ctx context.Context, hst host.Host,
	actionNameStateMap map[resources.Action]map[resources.Name]resources.State,
) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Mergeable.ConfigureAll", "actions", actionNameStateMap)
	funcCall := m.getFnCall()
	if funcCall == nil {
		MergeableT.Fatalf("no more calls expected, got ConfigureAll(%#v)", actionNameStateMap)
	}
	if funcCall.ConfigureAll == nil {
		MergeableT.Fatalf("unexpected call: got ConfigureAll(%#v), expected %#v", actionNameStateMap, funcCall)
	}
	if !reflect.DeepEqual(funcCall.ConfigureAll.ActionNameStateMap, actionNameStateMap) {
		MergeableT.Fatalf(
			"unexpected arguments: got ConfigureAll(%#v), expected ConfigureAll(%#v)",
			actionNameStateMap, funcCall.ConfigureAll.ActionNameStateMap,
		)
	}
	return funcCall.ConfigureAll.ReturnError
}

func SetupMergeableType(t *testing.T, individualFnCalls []MergeableFnCall) {
	MergeableT = t
	MergeableExpectedFnCalls = individualFnCalls
	t.Cleanup(func() {
		if len(MergeableExpectedFnCalls) > 0 {
			t.Errorf("expected calls pending:\n%v", MergeableExpectedFnCalls)
		}
	})
}

func init() {
	resources.MergeableResourcesTypeMap["Mergeable"] = Mergeable{}
	resources.ResourcesStateMap["Mergeable"] = MergeableState{}
}
