package resource

import (
	"context"

	hostPkg "github.com/fornellas/resonance/draft/host"
)

type APTPackageState struct {
	Name string `json:"name"`
}

func (f *APTPackageState) Id() *Id {
	return &Id{
		Type: FileType,
		Name: Name(f.Name),
	}
}

type APTPackageGroup struct{}

func (f *APTPackageGroup) Load(ctx context.Context, host hostPkg.Host, names []Name) (States, error) {
	aptPackageStates := make([]*APTPackageState, len(names))
	// TODO load to aptPackageStates
	states := make(States, len(names))
	for i, aptPackageState := range aptPackageStates {
		states[i] = aptPackageState
	}
	return states, nil
}

func (f *APTPackageGroup) Apply(ctx context.Context, host hostPkg.Host, states []State) error {
	aptPackageStates := make([]*APTPackageState, len(states))
	for i, state := range states {
		aptPackageStates[i] = state.(*APTPackageState)
	}
	// TODO apply aptPackageStates
	return nil
}

var APTPackageType = NewGroupType("APTPackage", &APTPackageGroup{})
