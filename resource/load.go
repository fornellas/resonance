package resource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
)

// ResourceName is the name of the resource.
// Must match resource's reflect.Type.Name().
type ResourceName string

// Resource returns an instance for the resource.
func (rn ResourceName) Resource() (Resource, error) {
	switch string(rn) {
	case "File":
		return File{}, nil
	default:
		return nil, fmt.Errorf("unknown resource type '%s'", rn)
	}
}

// ResourceDefinition groups Instance by ResourceName.
type ResourceDefinition struct {
	ResourceInstanceKey ResourceInstanceKey `yaml:"resource"`
	Parameters          ResourceParameters  `yaml:"parameters"`
}

// ResourceDefinitions is the schema used for loading resources from yaml files.
type ResourceDefinitions []ResourceDefinition

func (rd ResourceDefinitions) ReadState(ctx context.Context, host host.Host) (StateData, error) {
	stateData := StateData{}

	for _, resourceDefinition := range rd {
		resourceName, instanceName, err := resourceDefinition.ResourceInstanceKey.GetNames()
		if err != nil {
			return StateData{}, err
		}
		resource, err := resourceName.Resource()
		if err != nil {
			return StateData{}, err
		}
		instance := Instance{
			Name:       instanceName,
			Parameters: resourceDefinition.Parameters,
		}
		resourceState, err := resource.ReadState(ctx, host, instance)
		if err != nil {
			return StateData{}, fmt.Errorf("%s: failed to read state: %w", instanceName, err)
		}

		resourceInstanceKey := GetResourceInstanceKey(
			resourceName,
			instance.Name,
		)
		stateData[resourceInstanceKey] = resourceState
	}

	return stateData, nil
}

// Load loads resource definitions from given Yaml file path which contains
// the schema defined by ResourceDefinitions.
func LoadResourceDefinitions(ctx context.Context, path string) (ResourceDefinitions, error) {

	f, err := os.Open(path)
	if err != nil {
		return ResourceDefinitions{}, fmt.Errorf("failed to load resource definitions: %w", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	resourceDefinitions := ResourceDefinitions{}

	for {
		docResourceDefinitions := ResourceDefinitions{}
		if err := decoder.Decode(&docResourceDefinitions); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return ResourceDefinitions{}, fmt.Errorf("failed to load resource definitions: %s: %w", path, err)
		}
		resourceDefinitions = append(resourceDefinitions, docResourceDefinitions...)
	}

	return resourceDefinitions, nil
}
