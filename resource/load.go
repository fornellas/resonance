package resource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
)

// ResourceName is the name of the resource.
// Must match resource's reflect.Type.Name().
type ResourceName string

// Resource returns an instance for the resource.
func (rn ResourceName) Resource() (Resource, error) {
	for i := 0; i < reflect.TypeOf("").NumMethod(); i++ {
		typeObj := reflect.TypeOf("").Method(i).Type.Out(0)
		if typeObj.String() == string(rn) {
			newObj := reflect.New(typeObj).Elem().Interface()
			resource, ok := newObj.(Resource)
			if !ok {
				panic("Unable to cast to Resource")
			}
			return resource, nil
		}
	}
	return nil, fmt.Errorf("invalid resource type %s", rn)
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
		for resourceName, instances := range resourceDefinition {
			resource, err := resourceName.Resource()
			if err != nil {
				return StateData{}, err
			}
			resourceState, err := resource.ReadState(ctx, host, instances)
			if err != nil {
				return StateData{}, err
			}

			for _, instance := range instances {
				resourceInstanceKey := GetResourceInstanceKey(
					resourceName,
					instance.Name,
				)
				stateData[resourceInstanceKey] = resourceState
			}
		}
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
