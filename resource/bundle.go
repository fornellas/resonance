package resource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
				return nil, err
			}
			updatedAndValidatedDocResources = append(updatedAndValidatedDocResources, resource)
		}
		resources = append(resources, updatedAndValidatedDocResources...)
	}

	nestedLogger.WithField("", resources.String()).Trace("Resources")

	return resources, nil
}

// Bundle holds all resources for a host.
type Bundle []Resources

func (b Bundle) validate() error {
	resourceMap := map[TypeName]bool{}

	for _, resources := range b {
		for _, resource := range resources {
			if _, ok := resourceMap[resource.TypeName]; ok {
				return fmt.Errorf("duplicate resource %s", resource.TypeName)
			}
			resourceMap[resource.TypeName] = true
		}
	}
	return nil
}

// HasTypeName returns true if Resource is contained at Bundle.
func (b Bundle) HasTypeName(typeName TypeName) bool {
	for _, resources := range b {
		for _, resource := range resources {
			if resource.TypeName == typeName {
				return true
			}
		}
	}
	return false
}

// Resources returns all Resource at the bundle
func (b Bundle) Resources() Resources {
	allResources := Resources{}
	for _, resources := range b {
		allResources = append(allResources, resources...)
	}
	return allResources
}

// TypeNames returns all TypeName at the bundle
func (b Bundle) TypeNames() []TypeName {
	typeNames := []TypeName{}
	for _, resources := range b {
		for _, resource := range resources {
			typeNames = append(typeNames, resource.TypeName)
		}
	}
	return typeNames
}

// IsClean checks whether all resources at Bundle are clean.
func (b Bundle) IsClean(
	ctx context.Context,
	hst host.Host,
	typeNameStateMap TypeNameStateMap,
) (bool, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🕵️ Checking if state is clean")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	clean := true

	for _, resources := range b {
		for _, resource := range resources {
			currentState, ok := typeNameStateMap[resource.TypeName]
			if !ok {
				panic(fmt.Sprintf("TypeNameStateMap missing %s", resource.TypeName))
			}

			var chunks Chunks
			if resource.Destroy {
				chunks = Diff(currentState, nil)
			} else {
				chunks = Diff(currentState, resource.State)
			}

			if chunks.HaveChanges() {
				nestedLogger.WithField(
					"", chunks.String(),
				).Infof("%s %s", ActionConfigure.Emoji(), resource)
				clean = false
			} else {
				nestedLogger.Infof("%s %s", ActionOk.Emoji(), resource)
			}
		}
	}

	return clean, nil
}

// LoadBundle search for .yaml files at root, each having the Resources schema,
// loads and returns all of them.
// Bundle is sorted by alphabetical order.
func LoadBundle(ctx context.Context, hst host.Host, root string) (Bundle, error) {
	logger := log.GetLogger(ctx)
	logger.Infof("📂 Loading resources from %s", root)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	bundle := Bundle{}

	paths := []string{}
	if err := filepath.Walk(root, func(path string, fileInfo fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fileInfo.IsDir() || !strings.HasSuffix(fileInfo.Name(), ".yaml") {
			nestedLogger.Debugf("Skipping %s", path)
			return nil
		}
		nestedLogger.Debugf("Found resources file %s", path)
		paths = append(paths, path)
		return nil
	}); err != nil {
		return bundle, err
	}
	if len(paths) == 0 {
		return Bundle{}, fmt.Errorf("no .yaml resource files found under %s", root)
	}
	sort.Strings(paths)

	for _, path := range paths {
		resources, err := LoadResources(nestedCtx, hst, path)
		if err != nil {
			return bundle, err
		}
		bundle = append(bundle, resources)
	}

	if err := bundle.validate(); err != nil {
		return bundle, err
	}

	return bundle, nil
}
