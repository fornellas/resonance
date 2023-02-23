package resource

import (
	"context"
	"errors"
)

// ResourceName is the name of the resource.
// Must match resource's reflect.Type.Name().
type ResourceName string

// ResourceDefinition groups Instance by ResourceName.
type ResourceDefinition map[ResourceName][]Instance

// ResourceDefinitions is the schema used for loading resources from yaml files.
type ResourceDefinitions []ResourceDefinition

// Load loads resource definitions from given Yaml file path which contains
// the schema defined by ResourceDefinitions.
func Load(ctx context.Context, path string) (ResourceDefinitions, error) {
	return ResourceDefinitions{}, errors.New("TODO resource.Load")
}
