package resource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sirupsen/logrus"

	"github.com/fornellas/resonance/host"
)

// Name is a name that globally uniquely identifies a resource instance of a given type.
// Eg: for File type a Name would be the file absolute path such as /etc/issue.
type Name string

func (n Name) String() string {
	return string(n)
}

// Parameters for a resource instance. This is specific for each resource type.
// It must be unmarshallable by gopkg.in/yaml.v3.
// type Parameters yaml.Node

// Instance holds parameters for a resource instance.
type Instance struct {
	Name       Name      `yaml:"name"`
	Parameters yaml.Node `yaml:"parameters"`
}

// State holds information about a resource state. This is specific for each resource type.
// It must be marshallable by gopkg.in/yaml.v3.
// It must work with reflect.DeepEqual.
type State interface{}

// ManageableResource defines an interface for managing resource state.
type ManageableResource interface {
	// MergeApply informs whether all resources from the same type are to
	// be always merged together when applying.
	// When true, Apply is called only once, with all instances.
	// When false, Apply is called one time for each instance.
	MergeApply() bool

	// GetState returns current resource state from host without any side effects.
	GetState(ctx context.Context, host host.Host, name Name) (State, error)

	// GetDesiredState return desired state for given parameters.
	GetDesiredState(ctx context.Context, parameters yaml.Node) (State, error)

	// Apply confiugres the resource at host to given instances state.
	// Must be idempotent.
	Apply(ctx context.Context, host host.Host, instances []Instance) error

	// Destroy a configured resource at given host.
	// Must be idempotent.
	Destroy(ctx context.Context, host host.Host, name Name) error
}

// Type is the name of the resource.
// Must match resource's reflect.Type.Name().
type Type string

func (t Type) String() string {
	return string(t)
}

var TypeToManageableResource = map[Type]ManageableResource{}

// ManageableResource returns an instance for the resource.
func (t Type) ManageableResource() (ManageableResource, error) {
	manageableResource, ok := TypeToManageableResource[t]
	if !ok {
		types := []string{}
		for tpe := range TypeToManageableResource {
			types = append(types, tpe.String())
		}
		return nil, fmt.Errorf("unknown resource type '%s'; valid types: %s", t, strings.Join(types, ", "))
	}
	return manageableResource, nil
}

// TypeName is a string that identifies a resource type and name.
// Eg: File[/etc/issue].
type TypeName string

var resourceInstanceKeyRegexp = regexp.MustCompile(`^(.+)\[(.+)\]$`)

// GetTypeName returns the Type and Name.
func (rik TypeName) GetTypeName() (Type, Name, error) {
	var tpe Type
	var name Name
	matches := resourceInstanceKeyRegexp.FindStringSubmatch(string(rik))
	if len(matches) != 3 {
		return tpe, name, fmt.Errorf("%s does not match Type[Name] format", rik)
	}
	tpe = Type(matches[1])
	name = Name(matches[2])
	return tpe, name, nil
}

// NewTypeName creates a new TypeName.
func NewTypeName(tpe Type, name Name) TypeName {
	return TypeName(fmt.Sprintf("%s[%s]", tpe, name))
}

// HostState is the schema used to save/load state for all resources for a host.
type HostState map[TypeName]State

// Merge appends received HostState.
func (hs HostState) Merge(stateData HostState) {
	for resourceInstanceKey, resourceState := range stateData {
		if _, ok := hs[resourceInstanceKey]; ok {
			panic(fmt.Sprintf("duplicated resource instance %s", resourceInstanceKey))
		}
		hs[resourceInstanceKey] = resourceState
	}
}

func (hs HostState) String() (string, error) {
	buffer := bytes.Buffer{}
	encoder := yaml.NewEncoder(&buffer)
	if err := encoder.Encode(hs); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// ResourceDefinition is the schema used to declare a single resource within a file.
type ResourceDefinition struct {
	TypeName   TypeName  `yaml:"resource"`
	Parameters yaml.Node `yaml:"parameters"`
}

// ResourceBundle is the schema used to declare multiple resources at a single file.
type ResourceBundle []ResourceDefinition

// ResourceBundles holds all resources definitions for a host.
type ResourceBundles []ResourceBundle

// GetHostState reads and return the state from all resource definitions.
func (rbs ResourceBundles) GetHostState(ctx context.Context, host host.Host) (HostState, error) {
	hostState := HostState{}

	for _, resourceBundle := range rbs {
		for _, resourceDefinition := range resourceBundle {
			tpe, name, err := resourceDefinition.TypeName.GetTypeName()
			if err != nil {
				return hostState, err
			}
			resource, err := tpe.ManageableResource()
			if err != nil {
				return hostState, err
			}
			state, err := resource.GetState(ctx, host, name)
			if err != nil {
				return hostState, fmt.Errorf("%s: failed to read state: %w", resourceDefinition.TypeName, err)
			}

			hostState[resourceDefinition.TypeName] = state
		}
	}

	return hostState, nil
}

// GetDesiredHostState returns the desired HostState for all resources.
func (rbs ResourceBundles) GetDesiredHostState(ctx context.Context) (HostState, error) {
	hostState := HostState{}

	for _, resourceBundle := range rbs {
		for _, resourceDefinition := range resourceBundle {
			tpe, _, err := resourceDefinition.TypeName.GetTypeName()
			if err != nil {
				return hostState, err
			}
			resource, err := tpe.ManageableResource()
			if err != nil {
				return hostState, err
			}
			state, err := resource.GetDesiredState(ctx, resourceDefinition.Parameters)
			if err != nil {
				return hostState, fmt.Errorf("%s: failed get desired state: %w", resourceDefinition.TypeName, err)
			}

			hostState[resourceDefinition.TypeName] = state
		}
	}

	return hostState, nil
}

func loadResourceBundle(ctx context.Context, path string) (ResourceBundle, error) {
	f, err := os.Open(path)
	if err != nil {
		return ResourceBundle{}, fmt.Errorf("failed to load resource definitions: %w", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	resourceBundle := ResourceBundle{}

	for {
		docResourceBundle := ResourceBundle{}
		if err := decoder.Decode(&docResourceBundle); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return ResourceBundle{}, fmt.Errorf("failed to load resource definitions: %s: %w", path, err)
		}
		resourceBundle = append(resourceBundle, docResourceBundle...)
	}

	return resourceBundle, nil
}

// LoadResourceBundles loads resource definitions from all given Yaml file paths.
// Each file must have the schema defined by ResourceBundle.
func LoadResourceBundles(ctx context.Context, paths []string) ResourceBundles {
	resourceBundles := ResourceBundles{}
	for _, path := range paths {
		resourceBundle, err := loadResourceBundle(ctx, path)
		if err != nil {
			logrus.Fatal(err)
		}
		resourceBundles = append(resourceBundles, resourceBundle)
	}
	return resourceBundles
}
