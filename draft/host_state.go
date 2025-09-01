package draft

import (
	"context"
	"fmt"
	"reflect"

	"github.com/fornellas/resonance/host/types"
)

// State holds and manages the state of resources for a host.
type State struct {
	resourceTypeIdMap map[reflect.Type]map[string]Resource
	resources         []Resource
}

// Add given resource to State. Panics if a resource with same type and ID already exists.
func (s *State) MustAppendResource(resource Resource) {
	resourceType := reflect.TypeOf(resource)
	idMap, ok := s.resourceTypeIdMap[resourceType]
	if !ok {
		idMap = map[string]Resource{}
		s.resourceTypeIdMap[resourceType] = idMap
	}
	if _, ok := s.resourceTypeIdMap[resourceType][resource.ID()]; ok {
		panic(fmt.Sprintf("bug: duplicated resource: %T %v", resource, resource.ID()))
	}
	s.resourceTypeIdMap[resourceType][resource.ID()] = resource

	s.resources = append(s.resources, resource)
}

// Gets a Resource with same type and ID().
func (s *State) GetResourceByID(resource Resource) (Resource, bool) {
	resourceType := reflect.TypeOf(resource)
	idMap, ok := s.resourceTypeIdMap[resourceType]
	if ok {
		r, ok := idMap[resource.ID()]
		return r, ok
	}
	return nil, false
}

// Get a list of all resources.
func (s *State) GetResources() []Resource {
	return s.resources
}

// Applies the state of all resources to host.
func (s *State) Apply(ctx context.Context, host types.Host) error {
	var aptPackages APTPackages
	var dpkgArch *DpkgArch
	var files []*File

	for resourceType, resourceIdMap := range s.resourceTypeIdMap {
		switch resourceType {
		case reflect.TypeFor[*APTPackage]():
			for _, resource := range resourceIdMap {
				aptPackages = append(aptPackages, resource.(*APTPackage))
			}
		case reflect.TypeFor[*DpkgArch]():
			for _, resource := range resourceIdMap {
				dpkgArch = resource.(*DpkgArch)
			}
		case reflect.TypeFor[*File]():
			for _, resource := range resourceIdMap {
				files = append(files, resource.(*File))
			}
		default:
			panic(fmt.Sprintf("bug: unknown resource type: %T", resourceType))
		}
	}

	// We must first add extra dpkg archs, to enable APTPackages to work
	var preDpkgArch *DpkgArch
	if dpkgArch != nil {
		var err error
		preDpkgArch, err = LoadDpkgArch(ctx, host)
		if err != nil {
			return err
		}

		if err := preDpkgArch.Merge(dpkgArch); err != nil {
			return err
		}

		if err := preDpkgArch.Apply(ctx, host); err != nil {
			return err
		}
	}

	// Then we apply APTPackages, with present foreign dpkg archs.
	if len(aptPackages) > 0 {
		if err := aptPackages.Apply(ctx, host); err != nil {
			return err
		}
	}

	// Apply all files
	if len(files) > 0 {
		for _, file := range files {
			if err := file.Apply(ctx, host); err != nil {
				return err
			}
		}
	}

	// And finally apply the desired dpkg arch, which should remove required dpkg archs
	if dpkgArch != nil {
		ok, err := preDpkgArch.Satisfies(ctx, host, dpkgArch)
		if err != nil {
			return err
		}
		if !ok {
			if err := dpkgArch.Apply(ctx, host); err != nil {
				return err
			}
		}
	}

	return nil
}

// Load the full host state, for all resources.
func (s *State) Load(ctx context.Context, host types.Host) (*State, error) {
	loadedState := &State{}

	for resourceType, resourceIdMap := range s.resourceTypeIdMap {
		switch resourceType {
		case reflect.TypeFor[*APTPackage]():
			ids := []string{}
			for id := range resourceIdMap {
				ids = append(ids, id)
			}
			aptPackages, err := LoadAPTPackages(ctx, host, ids...)
			if err != nil {
				return nil, err
			}
			for _, aptPackage := range aptPackages {
				loadedState.MustAppendResource(aptPackage)
			}
		case reflect.TypeFor[*DpkgArch]():
			loadedDpkgArch, err := LoadDpkgArch(ctx, host)
			if err != nil {
				return nil, err
			}
			loadedState.MustAppendResource(loadedDpkgArch)
		case reflect.TypeFor[*File]():
			for id := range resourceIdMap {
				loadedFile, err := LoadFile(ctx, host, id)
				if err != nil {
					return nil, err
				}
				loadedState.MustAppendResource(loadedFile)
			}
		default:
			panic(fmt.Sprintf("bug: unknown resource type: %T", resourceType))
		}
	}

	return loadedState, nil
}

// Satisfies return true when self satisfies the state required by other.
func (s *State) Satisfies(ctx context.Context, host types.Host, otherState *State) (bool, error) {
	for _, otherResource := range otherState.GetResources() {
		resource, ok := s.GetResourceByID(otherResource)
		if !ok {
			return false, nil
		}
		ok, err := resource.Satisfies(ctx, host, otherResource)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}
