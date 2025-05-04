package resource

import (
	"context"
	"fmt"

	hostPkg "github.com/fornellas/resonance/draft/host"
)

// Single manages resources individually.
type Single interface {
	Load(ctx context.Context, host hostPkg.Host, name Name) (State, error)
	Apply(ctx context.Context, host hostPkg.Host, state State) error
}

type singleType struct {
	name   string
	single Single
}

func NewSingleType(name string, single Single) Type {
	return &singleType{
		name:   name,
		single: single,
	}
}

func (s *singleType) Name() string {
	return s.name
}

func (s *singleType) Single() Single {
	return s.single
}

func (*singleType) Group() Group {
	return nil
}

// Group manages resources in group.
type Group interface {
	// Load the state of all resources with given names from host.
	Load(ctx context.Context, host hostPkg.Host, names []Name) (States, error)
	// Apply given resource states to host.
	Apply(ctx context.Context, host hostPkg.Host, states []State) error
}

type groupType struct {
	name  string
	group Group
}

func NewGroupType(name string, group Group) Type {
	return &groupType{
		name:  name,
		group: group,
	}
}

func (g *groupType) Name() string {
	return g.name
}

func (*groupType) Single() Single {
	return nil
}

func (g *groupType) Group() Group {
	return g.group
}

// Type of a resource.
type Type interface {
	// Name of the type.
	Name() string
	// Single returns non-nil for resources that are to be managed one by one.
	Single() Single
	// Group returns non-nil for resources that are to be managed in group.
	Group() Group
}

// Name uniquely identify a resource at a host for the same Type.
type Name string

// Id uniquely identify a resource at a host, by its Type and Name.
type Id struct {
	Type Type
	Name Name
}

func (i *Id) String() string {
	return fmt.Sprintf("%s[%s]", i.Type.Name(), i.Name)
}

// State represents the state of a resource at a host.
type State interface {
	Id() *Id
}

type States []State

func (r States) HasId(*Id) bool {
	return false
}

func Load(ctx context.Context, host hostPkg.Host, ids []*Id) (States, error) {
	singleIds := []*Id{}
	typeToNamesMap := map[Type][]Name{}
	for _, id := range ids {
		if single := id.Type.Single(); single != nil {
			singleIds = append(singleIds, id)
		} else if group := id.Type.Group(); group != nil {
			typeToNamesMap[id.Type] = append(typeToNamesMap[id.Type], id.Name)
		} else {
			panic("bug: Type is neither Single nor Group")
		}
	}

	stateMap := map[string]State{}
	for _, id := range singleIds {
		var err error
		// TODO go routine
		stateMap[id.String()], err = id.Type.Single().Load(ctx, host, id.Name)
		if err != nil {
			return nil, err
		}
	}
	for tpe, names := range typeToNamesMap {
		// TODO go routine
		states, err := tpe.Group().Load(ctx, host, names)
		if err != nil {
			return nil, err
		}
		for _, state := range states {
			stateMap[state.Id().String()] = state
		}
	}

	states := make([]State, len(ids))
	for i, id := range ids {
		states[i] = stateMap[id.String()]
	}

	return states, nil
}

func Apply(context.Context, hostPkg.Host, States) error {
	// TODO apply single / group
	return nil
}
