package resource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

// Resources is the schema used to declare multiple resources at a single file.
type ResourceDefs []ResourceDef

func (rs ResourceDefs) Validate() error {
	resourceMap := map[TypeName]bool{}
	for _, resource := range rs {
		if _, ok := resourceMap[resource.TypeName]; ok {
			return fmt.Errorf("duplicate resource %s", resource.TypeName)
		}
		resourceMap[resource.TypeName] = true
	}
	return nil
}

func (rs ResourceDefs) TypeNames() []TypeName {
	typeNames := []TypeName{}
	for _, resource := range rs {
		typeNames = append(typeNames, resource.TypeName)
	}
	return typeNames
}

func (rs ResourceDefs) LogValue() slog.Value {
	bs, err := yaml.Marshal(rs)
	if err != nil {
		panic(err)
	}
	return slog.StringValue(strings.Trim(string(bs), "\n"))
}

// LoadFile loads Resources declared at given YAML file path
func LoadFile(ctx context.Context, hst host.Host, path string) (ResourceDefs, error) {
	ctx, logger := log.MustContextLoggerIndented(ctx)

	logger.Info("📂 Loading Yaml", "path", path)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load resource file: %w", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	resourceDefs := ResourceDefs{}

	for {
		docResources := ResourceDefs{}
		if err := decoder.Decode(&docResources); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return ResourceDefs{}, fmt.Errorf("failed to load resource file: %s: %w", path, err)
		}
		if err := docResources.Validate(); err != nil {
			return ResourceDefs{}, fmt.Errorf("resource file validation failed: %s: %w", path, err)
		}
		updatedAndValidatedDocResources := ResourceDefs{}
		for _, resource := range docResources {
			var err error
			resource.State, err = resource.State.ValidateAndUpdate(ctx, hst)
			if err != nil {
				return nil, fmt.Errorf("%s: %s: %w", path, resource, err)
			}
			updatedAndValidatedDocResources = append(updatedAndValidatedDocResources, resource)
		}
		resourceDefs = append(resourceDefs, updatedAndValidatedDocResources...)
	}

	if len(resourceDefs) == 0 {
		return resourceDefs, fmt.Errorf("file has no declared resources: %s", path)
	}

	logger = log.MustLoggerIndented(ctx)
	logger.Debug("Details", "resources", resourceDefs)

	return resourceDefs, nil
}

func findYmls(ctx context.Context, root string) ([]string, error) {
	logger := log.MustLoggerIndented(ctx)

	yamlPaths := []string{}
	if err := filepath.Walk(root, func(path string, fileInfo fs.FileInfo, err error) error {
		logger := logger.With("path", path)
		if err != nil {
			return err
		}
		if fileInfo.IsDir() || !strings.HasSuffix(fileInfo.Name(), ".yaml") {
			logger.Debug("Skipping")
			return nil
		}
		logger.Debug("Found resources Yaml")
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
func LoadDir(ctx context.Context, hst host.Host, root string) (ResourceDefs, error) {
	ctx, logger := log.MustContextLoggerIndented(ctx)

	logger.Info("📂 Loading resources")

	resourceDefs := ResourceDefs{}

	yamlPaths, err := findYmls(ctx, root)
	if err != nil {
		return resourceDefs, err
	}

	for _, yamlPath := range yamlPaths {
		yamlResources, err := LoadFile(ctx, hst, yamlPath)
		if err != nil {
			return resourceDefs, err
		}
		resourceDefs = append(resourceDefs, yamlResources...)
	}

	if err := resourceDefs.Validate(); err != nil {
		return resourceDefs, err
	}

	return resourceDefs, nil
}
