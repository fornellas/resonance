package resource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
)

// InstanceName is a name that globally uniquely identifies a resource instance.
type InstanceName string

// Parameters for a resource instance.
// It must be unmarshallable by gopkg.in/yaml.v3.
type Parameters interface{}

// Instance holds parameters for a resource instance.
type Instance struct {
	Name       InstanceName `yaml:"name"`
	Parameters Parameters   `yaml:"parameters"`
}

// State holds information about a resource state.
// It must be marshallable by gopkg.in/yaml.v3.
// It must work with reflect.DeepEqual.
type State interface{}

// ManageableResource defines an interface for managing resource state.
type ManageableResource interface {
	// AlwaysMergeApply informs whether all resources from the same type are to
	// be always merged together when applying.
	// When true, Apply is called only once, with all instances.
	// When false, Apply is called one time for each instance.
	AlwaysMergeApply() bool

	// Reads current resource state without any side effects.
	ReadState(
		ctx context.Context,
		host host.Host,
		instance Instance,
	) (State, error)

	// Apply confiugres the resource at host to given instances state.
	// Must be idempotent.
	Apply(
		ctx context.Context,
		host host.Host,
		instances []Instance,
	) error

	// Destroy a configured resource at given host.
	// Must be idempotent.
	Destroy(
		ctx context.Context,
		host host.Host,
		instances []Instance,
	) error
}

////////////////////////////////////////////
////////////////////////////////////////////
////////////////////////////////////////////
////////////////////////////////////////////
////////////////////////////////////////////

// ResourceName is the name of the resource.
// Must match resource's reflect.Type.Name().
type ResourceName string

// ManageableResource returns an instance for the resource.
func (rn ResourceName) ManageableResource() (ManageableResource, error) {
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
	Parameters          Parameters          `yaml:"parameters"`
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
		resource, err := resourceName.ManageableResource()
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

type ResourceInstanceKey string

var resourceInstanceKeyRegexp = regexp.MustCompile(`^(.+)\[(.+)\]$`)

func (rik ResourceInstanceKey) GetNames() (ResourceName, InstanceName, error) {
	var resourceName ResourceName
	var instanceName InstanceName
	matches := resourceInstanceKeyRegexp.FindStringSubmatch(string(rik))
	if len(matches) != 3 {
		return resourceName, instanceName, fmt.Errorf("%s does not match Type[Name] format", rik)
	}
	resourceName = ResourceName(matches[1])
	instanceName = InstanceName(matches[2])
	return resourceName, instanceName, nil
}

func GetResourceInstanceKey(resourceName ResourceName, instanceName InstanceName) ResourceInstanceKey {
	return ResourceInstanceKey(fmt.Sprintf("%s[%s]", resourceName, instanceName))
}

type StateData map[ResourceInstanceKey]State

func (sd StateData) Merge(stateData StateData) {
	for resourceInstanceKey, resourceState := range stateData {
		if _, ok := sd[resourceInstanceKey]; ok {
			panic(fmt.Sprintf("duplicated resource instance %s", resourceInstanceKey))
		}
		sd[resourceInstanceKey] = resourceState
	}
}
