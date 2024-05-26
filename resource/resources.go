package resource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

// Resources is the schema used to declare multiple resources at a single file.
type Resources []Resource

func (rs Resources) Validate() error {
	resourceMap := map[TypeName]bool{}
	for _, resource := range rs {
		if _, ok := resourceMap[resource.TypeName]; ok {
			return fmt.Errorf("duplicate resource %s", resource.TypeName)
		}
		resourceMap[resource.TypeName] = true
	}
	return nil
}

func (rs Resources) Len() int {
	return len(rs)
}

func (rs Resources) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}

func (rs Resources) Less(i, j int) bool {
	return rs[i].String() < rs[j].String()
}

func (rs Resources) TypeNames() []TypeName {
	typeNames := []TypeName{}
	for _, resource := range rs {
		typeNames = append(typeNames, resource.TypeName)
	}
	return typeNames
}

func (rs Resources) String() string {
	bytes, err := yaml.Marshal(&rs)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

// LoadBundle loads resources from given Yaml file path.
func LoadResources(ctx context.Context, hst host.Host, path string) (Resources, error) {
	logger := log.GetLogger(ctx)
	logger.Infof("%s", path)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load resource file: %w", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	resources := Resources{}

	for {
		docResources := Resources{}
		if err := decoder.Decode(&docResources); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return Resources{}, fmt.Errorf("failed to load resource file: %s: %w", path, err)
		}
		if err := docResources.Validate(); err != nil {
			return Resources{}, fmt.Errorf("resource file validation failed: %s: %w", path, err)
		}
		updatedAndValidatedDocResources := Resources{}
		for _, resource := range docResources {
			var err error
			resource.State, err = resource.State.ValidateAndUpdate(nestedCtx, hst)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", resource, err)
			}
			updatedAndValidatedDocResources = append(updatedAndValidatedDocResources, resource)
		}
		resources = append(resources, updatedAndValidatedDocResources...)
	}

	nestedLogger.WithField("", resources.String()).Trace("Resources")

	return resources, nil
}
