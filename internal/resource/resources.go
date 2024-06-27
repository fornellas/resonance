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

func (rs Resources) validate() error {
	resourceMap := map[TypeName]bool{}

	for _, resource := range rs {
		if _, ok := resourceMap[resource.TypeName]; ok {
			return fmt.Errorf("duplicate resource %s", resource.TypeName)
		}
		resourceMap[resource.TypeName] = true
	}
	return nil
}

// LoadFile loads Resources declared at given YAML file path
func LoadFile(ctx context.Context, hst host.Host, yamlPath string) (Resources, error) {
	logger := log.GetLogger(ctx)
	logger.Infof("%s", yamlPath)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	f, err := os.Open(yamlPath)
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
			return Resources{}, fmt.Errorf("failed to load resource file: %s: %w", yamlPath, err)
		}
		if err := docResources.Validate(); err != nil {
			return Resources{}, fmt.Errorf("resource file validation failed: %s: %w", yamlPath, err)
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

// LoadDir search for .yaml files at root and loads all of them.
// Files are sorted by alphabetical order.
func LoadDir(ctx context.Context, hst host.Host, root string) (Resources, error) {
	logger := log.GetLogger(ctx)
	logger.Infof("📂 Loading resources from %s", root)
	nestedCtx := log.IndentLogger(ctx)

	resources := Resources{}

	yamlPaths, err := findYmls(nestedCtx, root)
	if err != nil {
		return resources, err
	}

	for _, yamlPath := range yamlPaths {
		yamlResources, err := LoadFile(nestedCtx, hst, yamlPath)
		if err != nil {
			return resources, err
		}
		resources = append(resources, yamlResources...)
	}

	if err := resources.validate(); err != nil {
		return resources, err
	}

	return resources, nil
}
