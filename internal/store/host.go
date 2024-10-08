package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
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
	Host                  host.Host
	basePath              string
	originalResourcesPath string
}

// NewHostStore creates a new HostStore for given Host.
func NewHostStore(hst host.Host, path string) *HostStore {
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

func (s *HostStore) SaveOriginalResource(ctx context.Context, resource resourcesPkg.Resource) error {
	path := s.getOriginalResourcePath(resource)

	hasFile, err := s.hasFile(ctx, path)
	if err != nil {
		return err
	}

	var resourceStore resourceStoreSchema
	if hasFile {
		resourceStoreBytes, err := s.Host.ReadFile(ctx, path)
		if err != nil {
			return err
		}

		resourceStore = resourceStoreSchema{}
		err = yaml.Unmarshal(resourceStoreBytes, &resourceStore)
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

	resourceStoreBytes, err := s.Host.ReadFile(ctx, path)
	if err != nil {
		return false, err
	}
	resourceStore := resourceStoreSchema{}
	err = yaml.Unmarshal(resourceStoreBytes, &resourceStore)
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

	resourceStoreBytes, err := s.Host.ReadFile(ctx, path)
	if err != nil {
		return nil, err
	}

	resourceStore := resourceStoreSchema{}
	err = yaml.Unmarshal(resourceStoreBytes, &resourceStore)
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
		resourceStoreBytes, err := s.Host.ReadFile(ctx, path)
		if err != nil {
			return err
		}

		resourceStore = resourceStoreSchema{}
		err = yaml.Unmarshal(resourceStoreBytes, &resourceStore)
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
	blueprintBytes, err := yaml.Marshal(obj)
	if err != nil {
		panic(fmt.Sprintf("bug: failed to serialize blueprint: %s", err.Error()))
	}

	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, fs.FileMode(0700)); err != nil {
		return err
	}

	return s.Host.WriteFile(ctx, path, blueprintBytes, 0600)
}

func (s *HostStore) SaveLastBlueprint(ctx context.Context, blueprint *blueprintPkg.Blueprint) error {
	path := s.getBlueprintPath("last")
	return s.saveYaml(ctx, blueprint, path)
}

func (s *HostStore) loadBlueprint(ctx context.Context, name string) (*blueprintPkg.Blueprint, error) {
	path := s.getBlueprintPath(name)

	blueprintBytes, err := s.Host.ReadFile(ctx, path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	blueprint := &blueprintPkg.Blueprint{}
	err = yaml.Unmarshal(blueprintBytes, blueprint)
	if err != nil {
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
