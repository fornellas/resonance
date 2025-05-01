package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"

	blueprintPkg "github.com/fornellas/resonance/blueprint"
	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

type resourceStoreSchema map[string]map[string]resourcesPkg.Resource

func (s *resourceStoreSchema) UnmarshalYAML(node *yaml.Node) error {
	type UnmarshalSchema map[string]map[string]yaml.Node

	resourceStore := UnmarshalSchema{}

	node.KnownFields(true)
	err := node.Decode(&resourceStore)
	if err != nil {
		return fmt.Errorf("line %d: %s", node.Line, err.Error())
	}

	for typeName, idResourceMap := range resourceStore {
		(*s)[typeName] = map[string]resourcesPkg.Resource{}
		for id, resourceNode := range idResourceMap {
			resource := resourcesPkg.GetResourceByTypeName(typeName)
			if resource == nil {
				return fmt.Errorf("line %d: invalid single resource type: %#v ", resourceNode.Line, typeName)
			}
			resourceNode.KnownFields(true)
			err := resourceNode.Decode(resource)
			if err != nil {
				return fmt.Errorf("line %d: %s", resourceNode.Line, err.Error())
			}
			(*s)[typeName][id] = resource
		}
	}

	return nil
}

// Implementation of Store that persists Blueprints at a Host at Path.
type HostStore struct {
	Host                  types.Host
	basePath              string
	originalResourcesPath string
}

// NewHostStore creates a new HostStore for given Host.
func NewHostStore(hst types.Host, path string) *HostStore {
	// prefix the store path with a DB version, so we can handle changes in the store format
	storePath := filepath.Join(path, "v1")
	return &HostStore{
		Host:                  hst,
		basePath:              storePath,
		originalResourcesPath: filepath.Join(storePath, "original"),
	}
}

func (s *HostStore) hasFile(ctx context.Context, path string) (bool, error) {
	_, err := s.Host.Lstat(ctx, path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *HostStore) getOriginalResourcePath(resource resourcesPkg.Resource) string {
	hash := resourcesPkg.HashResource(resource)
	return fmt.Sprintf("%s.yaml", filepath.Join(s.originalResourcesPath, hash))
}

func (s *HostStore) loadResourceStore(ctx context.Context, path string) (resourceStoreSchema, error) {
	resourceStoreReadCloser, err := s.Host.ReadFile(ctx, path)
	if err != nil {
		return nil, err
	}

	yamlDecoder := yaml.NewDecoder(resourceStoreReadCloser)
	yamlDecoder.KnownFields(true)

	resourceStore := resourceStoreSchema{}
	err = yamlDecoder.Decode(&resourceStore)
	if err != nil {
		if closeErr := resourceStoreReadCloser.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return nil, err
	}
	if closeErr := resourceStoreReadCloser.Close(); closeErr != nil {
		return nil, err
	}
	return resourceStore, nil
}

func (s *HostStore) SaveOriginalResource(ctx context.Context, resource resourcesPkg.Resource) error {
	path := s.getOriginalResourcePath(resource)

	hasFile, err := s.hasFile(ctx, path)
	if err != nil {
		return err
	}

	var resourceStore resourceStoreSchema
	if hasFile {
		resourceStore, err = s.loadResourceStore(ctx, path)
		if err != nil {
			return err
		}

		resourceStore[resourcesPkg.GetResourceTypeName(resource)] = map[string]resourcesPkg.Resource{
			resourcesPkg.GetResourceId(resource): resource,
		}
	} else {
		resourceStore = resourceStoreSchema{}
	}

	resourceStore[resourcesPkg.GetResourceTypeName(resource)] = map[string]resourcesPkg.Resource{
		resourcesPkg.GetResourceId(resource): resource,
	}

	return s.saveYaml(ctx, resourceStore, path)
}

func (s *HostStore) HasOriginalResource(ctx context.Context, resource resourcesPkg.Resource) (bool, error) {
	path := s.getOriginalResourcePath(resource)

	hasFile, err := s.hasFile(ctx, path)
	if err != nil {
		return false, err
	}
	if !hasFile {
		return false, nil
	}

	resourceStore, err := s.loadResourceStore(ctx, path)
	if err != nil {
		return false, err
	}

	resourceTypeMap, ok := resourceStore[resourcesPkg.GetResourceTypeName(resource)]
	if !ok {
		return false, nil
	}

	_, ok = resourceTypeMap[resourcesPkg.GetResourceId(resource)]
	return ok, nil
}

func (s *HostStore) LoadOriginalResource(ctx context.Context, resource resourcesPkg.Resource) (resourcesPkg.Resource, error) {
	path := s.getOriginalResourcePath(resource)

	resourceStore, err := s.loadResourceStore(ctx, path)
	if err != nil {
		return nil, err
	}

	resourceTypeMap, ok := resourceStore[resourcesPkg.GetResourceTypeName(resource)]
	if !ok {
		return nil, errors.New("original resource not found")
	}

	originalResource, ok := resourceTypeMap[resourcesPkg.GetResourceId(resource)]
	if !ok {
		return nil, errors.New("original resource not found")
	}

	return originalResource, nil
}

func (s *HostStore) DeleteOriginalResource(ctx context.Context, resource resourcesPkg.Resource) error {
	path := s.getOriginalResourcePath(resource)

	hasFile, err := s.hasFile(ctx, path)
	if err != nil {
		return err
	}

	var resourceStore resourceStoreSchema
	if hasFile {
		resourceStore, err := s.loadResourceStore(ctx, path)
		if err != nil {
			return err
		}

		resourceStore[resourcesPkg.GetResourceTypeName(resource)] = map[string]resourcesPkg.Resource{
			resourcesPkg.GetResourceId(resource): resource,
		}
	} else {
		return nil
	}

	delete(resourceStore, resourcesPkg.GetResourceTypeName(resource))

	return s.saveYaml(ctx, resourceStore, path)
}

func (s *HostStore) getBlueprintPath(name string) string {
	return filepath.Join(s.basePath, fmt.Sprintf("%s.yaml", name))
}

func (s *HostStore) saveYaml(ctx context.Context, obj any, path string) error {
	dir := filepath.Dir(path)

	if err := lib.MkdirAll(ctx, s.Host, dir, 0700); err != nil {
		return err
	}

	var wg sync.WaitGroup

	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		return err
	}

	wg.Add(1)
	var encodeErr error
	var pipeWriterCloseErr error
	go func() {
		defer wg.Done()
		defer func() {
			pipeWriterCloseErr = pipeWriter.Close()
		}()
		encoder := yaml.NewEncoder(pipeWriter)
		encodeErr = encoder.Encode(obj)
	}()

	wg.Add(1)
	var writeFileErr error
	go func() {
		defer wg.Done()
		writeFileErr = s.Host.WriteFile(ctx, path, pipeReader, 0600)
	}()

	wg.Wait()

	return errors.Join(encodeErr, pipeWriterCloseErr, writeFileErr)
}

func (s *HostStore) SaveLastBlueprint(ctx context.Context, blueprint *blueprintPkg.Blueprint) error {
	path := s.getBlueprintPath("last")
	return s.saveYaml(ctx, blueprint, path)
}

func (s *HostStore) loadBlueprint(ctx context.Context, name string) (*blueprintPkg.Blueprint, error) {
	path := s.getBlueprintPath(name)

	blueprintReadCloser, err := s.Host.ReadFile(ctx, path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	yamlDecoder := yaml.NewDecoder(blueprintReadCloser)
	yamlDecoder.KnownFields(true)

	blueprint := &blueprintPkg.Blueprint{}
	err = yamlDecoder.Decode(blueprint)
	if err != nil {
		if closeErr := blueprintReadCloser.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return nil, err
	}
	if closeErr := blueprintReadCloser.Close(); closeErr != nil {
		return nil, err
	}
	return blueprint, nil
}

func (s *HostStore) LoadLastBlueprint(ctx context.Context) (*blueprintPkg.Blueprint, error) {
	return s.loadBlueprint(ctx, "last")
}

func (s *HostStore) SaveTargetBlueprint(ctx context.Context, blueprint *blueprintPkg.Blueprint) error {
	path := s.getBlueprintPath("target")
	return s.saveYaml(ctx, blueprint, path)
}

func (s *HostStore) HasTargetBlueprint(ctx context.Context) (bool, error) {
	path := s.getBlueprintPath("target")
	return s.hasFile(ctx, path)
}

func (s *HostStore) LoadTargetBlueprint(ctx context.Context) (*blueprintPkg.Blueprint, error) {
	return s.loadBlueprint(ctx, "target")
}

func (s *HostStore) DeleteTargetBlueprint(ctx context.Context) error {
	path := s.getBlueprintPath("target")
	err := s.Host.Remove(ctx, path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}

func (s *HostStore) GetLogWriterCloser(name string) io.WriteCloser {
	panic("TODO")
}
