package resource

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

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
				chunks = DiffResourceState(resource.ManageableResource(), currentState, nil)
			} else {
				chunks = DiffResourceState(resource.ManageableResource(), currentState, resource.State)
			}

			if chunks.HaveChanges() {
				nestedLogger.WithField(
					"", chunks.String(),
				).Infof("%s %s", ActionConfigure.Emoji(), resource)
				clean = false
			} else {
				nestedLogger.Debugf("%s %s", ActionOk.Emoji(), resource)
			}
		}
	}

	return clean, nil
}

func findYmls(ctx context.Context, root string) ([]string, error) {
	logger := log.GetLogger(ctx)

	yamlPaths := []string{}
	if err := filepath.Walk(root, func(path string, fileInfo fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fileInfo.IsDir() || !strings.HasSuffix(fileInfo.Name(), ".yaml") {
			logger.Debugf("Skipping %s", path)
			return nil
		}
		logger.Debugf("Found resources file %s", path)
		yamlPaths = append(yamlPaths, path)
		return nil
	}); err != nil {
		return nil, err
	}
	if len(yamlPaths) == 0 {
		return nil, fmt.Errorf("no .yaml resource files found under %s", root)
	}

	sort.Strings(yamlPaths)

	return yamlPaths, nil
}

// LoadBundle search for .yaml files at root, each having the Resources schema,
// loads and returns all of them.
// Bundle is sorted by alphabetical order.
func LoadBundle(ctx context.Context, hst host.Host, root string) (Bundle, error) {
	logger := log.GetLogger(ctx)
	logger.Infof("📂 Loading resources from %s", root)
	nestedCtx := log.IndentLogger(ctx)

	bundle := Bundle{}

	yamlPaths, err := findYmls(nestedCtx, root)
	if err != nil {
		return bundle, err
	}

	for _, yamlPath := range yamlPaths {
		resources, err := LoadResources(nestedCtx, hst, yamlPath)
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
