package resources

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

	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

func LoadFile(ctx context.Context, path string) (resourcesPkg.Resources, error) {
	logger := log.MustLogger(ctx).With("path", path)
	logger.Info("📝 Loading file")

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load resource file: %w", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	resources := resourcesPkg.Resources{}

	for {
		type ResourcesYaml []yaml.Node

		resourcesYaml := ResourcesYaml{}
		if err := decoder.Decode(&resourcesYaml); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to load resource file: %s: %w", path, err)
		}

		for _, resourceMapNode := range resourcesYaml {
			resourceMap := map[string]yaml.Node{}
			resourceMapNode.KnownFields(true)
			if err := resourceMapNode.Decode(resourceMap); err != nil {
				return nil, fmt.Errorf("failed to load resource file: %s:%d: %s", path, resourceMapNode.Line, err.Error())
			}
			if len(resourceMap) != 1 {
				return nil, fmt.Errorf("failed to load resource file: %s:%d: mapping must have a single key with the resource type", path, resourceMapNode.Line)
			}

			var resource resourcesPkg.Resource = nil
			for typeName, resourceNode := range resourceMap {
				if resource != nil {
					panic("bug: resource is not nil")
				}
				resource = resourcesPkg.GetResourceByName(typeName)
				if resource == nil {
					return nil, fmt.Errorf("failed to load resource file: %s:%d: invalid resource type %#v; valid types: %s", path, resourceMapNode.Line, typeName, strings.Join(resourcesPkg.GetResourceNames(), ", "))
				}

				resourceNode.KnownFields(true)
				if err := resourceNode.Decode(resource); err != nil {
					return nil, fmt.Errorf("failed to load resource file: %s:%d: %s", path, resourceMapNode.Line, err.Error())
				}

				if err := resource.Validate(); err != nil {
					return nil, fmt.Errorf("failed to load resource file: %s:%d: %s", path, resourceMapNode.Line, err.Error())
				}
			}
			if resource == nil {
				panic("bug: resource is nil")
			}

			resources = append(resources, resource)
		}
	}

	if len(resources) == 0 {
		return nil, fmt.Errorf("failed to load resource file: no resources found")
	}

	return resources, nil
}

func LoadDir(ctx context.Context, dir string) (resourcesPkg.Resources, error) {
	log.MustLogger(ctx).Info("📂 Loading directory", "dir", dir)
	ctx, logger := log.MustContextLoggerIndented(ctx)

	resources := resourcesPkg.Resources{}

	paths := []string{}
	if err := filepath.Walk(dir, func(path string, fileInfo fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fileInfo.IsDir() || !strings.HasSuffix(fileInfo.Name(), ".yaml") {
			logger.Debug("Skipping", "path", path)
			return nil
		}
		logger.Debug("Found resources file", "path", path)
		paths = append(paths, path)
		return nil
	}); err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no .yaml resource files found under %s", dir)
	}
	sort.Strings(paths)

	for _, path := range paths {
		fileResources, err := LoadFile(ctx, path)
		if err != nil {
			return nil, err
		}
		resources = append(resources, fileResources...)
	}

	if err := resources.Validate(); err != nil {
		return resources, err
	}

	return resources, nil
}
