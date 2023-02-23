package state

import (
	"context"
	"fmt"
	"reflect"
)

// ResourceState holds information about a resource.
// It must be marshallable by gopkg.in/yaml.v3.
// It must work with reflect.DeepEqual.
type ResourceState interface{}

type ResourceStateKey struct {
	ResourceType reflect.Type
	Name         string
}

func (sk ResourceStateKey) String() string {
	return fmt.Sprintf("%s[%s]", sk.ResourceType, sk.Name)
}

type StateData map[ResourceStateKey]ResourceState

type PersistantState interface {
	Load(ctx context.Context) (StateData, error)
	Save(ctx context.Context, stateData StateData) error
}
