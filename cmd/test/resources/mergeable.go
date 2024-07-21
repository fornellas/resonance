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

type MergeableFuncValidateName struct {
	Name        resources.Name
	ReturnError error
}

func (mfvn MergeableFuncValidateName) String() string {
	return fmt.Sprintf("(%#v) %#v", mfvn.Name, mfvn.ReturnError)
}

type MergeableFuncGetStates struct {
	Names              resources.Names
	ReturnNameStateMap map[resources.Name]resources.State
	ReturnError        error
}

func (mfgs MergeableFuncGetStates) String() string {
	return fmt.Sprintf("(%#v) (%#v, %#v)", mfgs.Names, mfgs.ReturnNameStateMap, mfgs.ReturnError)
}

type MergeableFuncApplyMerged struct {
	ActionNameStateMap map[resources.Action]map[resources.Name]resources.State
	ReturnError        error
}

func (mfca MergeableFuncApplyMerged) String() string {
	return fmt.Sprintf("(%#v) (%#v)", mfca.ActionNameStateMap, mfca.ReturnError)
}

type MergeableFuncDestroy struct {
	Name        resources.Name
	ReturnError error
}

func (ifd MergeableFuncDestroy) String() string {
	return fmt.Sprintf("(%#v) (%#v)", ifd.Name, ifd.ReturnError)
}

type MergeableFuncCall struct {
	ValidateName *MergeableFuncValidateName
	GetStates    *MergeableFuncGetStates
	ApplyMerged  *MergeableFuncApplyMerged
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

func (m Mergeable) ValidateName(name resources.Name) error {
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
	names resources.Names,
) (map[resources.Name]resources.State, error) {
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

func (m Mergeable) ApplyMerged(
	ctx context.Context, hst host.Host,
	actionNameStateMap map[resources.Action]map[resources.Name]resources.State,
) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Test.ApplyMerged(%#v)", actionNameStateMap)
	funcCall := m.getFuncCall()
	if funcCall == nil {
		MergeableT.Fatalf("no more calls expected, got ApplyMerged(%#v)", actionNameStateMap)
	}
	if funcCall.ApplyMerged == nil {
		MergeableT.Fatalf("unexpected call: got ApplyMerged(%#v), expected %#v", actionNameStateMap, funcCall)
	}
	if !reflect.DeepEqual(funcCall.ApplyMerged.ActionNameStateMap, actionNameStateMap) {
		MergeableT.Fatalf(
			"unexpected arguments: got ApplyMerged(%#v), expected ApplyMerged(%#v)",
			actionNameStateMap, funcCall.ApplyMerged.ActionNameStateMap,
		)
	}
	return funcCall.ApplyMerged.ReturnError
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
	resources.MergeableResourcesTypeMap["Mergeable"] = Mergeable{}
	resources.ResourcesStateMap["Mergeable"] = MergeableState{}
}
