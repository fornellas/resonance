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

type MergeableState struct {
	Value string
}

func (ms MergeableState) ValidateAndUpdate(ctx context.Context, hst host.Host) (resource.State, error) {
	return ms, nil
}

type MergeableFuncValidateName struct {
	Name        resource.Name
	ReturnError error
}

func (mfvn MergeableFuncValidateName) String() string {
	return fmt.Sprintf("(%#v) %#v", mfvn.Name, mfvn.ReturnError)
}

type MergeableFuncGetStates struct {
	Names              []resource.Name
	ReturnNameStateMap map[resource.Name]resource.State
	ReturnError        error
}

func (mfgs MergeableFuncGetStates) String() string {
	return fmt.Sprintf("(%#v) (%#v, %#v)", mfgs.Names, mfgs.ReturnNameStateMap, mfgs.ReturnError)
}

type MergeableFuncConfigureAll struct {
	ActionNameStateMap map[resource.Action]map[resource.Name]resource.State
	ReturnError        error
}

func (mfca MergeableFuncConfigureAll) String() string {
	return fmt.Sprintf("(%#v) (%#v)", mfca.ActionNameStateMap, mfca.ReturnError)
}

type MergeableFuncDestroy struct {
	Name        resource.Name
	ReturnError error
}

func (ifd MergeableFuncDestroy) String() string {
	return fmt.Sprintf("(%#v) (%#v)", ifd.Name, ifd.ReturnError)
}

type MergeableFuncCall struct {
	ValidateName *MergeableFuncValidateName
	GetStates    *MergeableFuncGetStates
	ConfigureAll *MergeableFuncConfigureAll
}

func (mfc MergeableFuncCall) String() string {
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
	panic("empty MergeableFuncCall")
}

type MergeableFuncCalls []MergeableFuncCall

func (mfcs MergeableFuncCalls) String() string {
	var s strings.Builder
	for i, mfc := range mfcs {
		fmt.Fprintf(&s, "%d: %v", i, mfc)
	}
	return s.String()
}

var MergeableT *testing.T
var MergeableExpectedFuncCalls MergeableFuncCalls

type Mergeable struct{}

func (m *Mergeable) getFuncCall() *MergeableFuncCall {
	if len(MergeableExpectedFuncCalls) == 0 {
		return nil
	}
	testFuncCall, expectedFuncCalls := MergeableExpectedFuncCalls[0], MergeableExpectedFuncCalls[1:]
	MergeableExpectedFuncCalls = expectedFuncCalls
	return &testFuncCall
}

func (m Mergeable) ValidateName(name resource.Name) error {
	funcCall := m.getFuncCall()
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
	names []resource.Name,
) (map[resource.Name]resource.State, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Test.GetStates(%#v)", names)
	funcCall := m.getFuncCall()
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
	actionNameStateMap map[resource.Action]map[resource.Name]resource.State,
) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Test.ConfigureAll(%#v)", actionNameStateMap)
	funcCall := m.getFuncCall()
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

func SetupMergeableType(t *testing.T, individualFuncCalls []MergeableFuncCall) {
	MergeableT = t
	MergeableExpectedFuncCalls = individualFuncCalls
	t.Cleanup(func() {
		if len(MergeableExpectedFuncCalls) > 0 {
			t.Errorf("expected calls pending:\n%v", MergeableExpectedFuncCalls)
		}
	})
}

func init() {
	resource.MergeableManageableResourcesTypeMap["Mergeable"] = Mergeable{}
	resource.ManageableResourcesStateMap["Mergeable"] = MergeableState{}
}
