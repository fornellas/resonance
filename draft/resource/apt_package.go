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
	// TODO
	return nil, nil
}

func (f *APTPackageGroup) Apply(ctx context.Context, host hostPkg.Host, states []State) error {
	// aptPackageState := state.(*APTPackageState)
	// TODO
	return nil
}

var APTPackageType = NewGroupType("APTPackage", &APTPackageGroup{})
