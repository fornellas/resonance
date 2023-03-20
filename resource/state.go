package resource

import (
	"context"
	"fmt"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/version"
)

// State is a Type specific interface for defining resource state as configured by users.
type State interface {
	// ValidateAndUpdate validates and updates the state with any required information from the host.
	// Eg: transform username into UID.
	ValidateAndUpdate(ctx context.Context, hst host.Host) (State, error)
}

// HostState holds the state for a host
type HostState struct {
	// Version of the binary used to put the host in this state.
	Version        version.Version `yaml:"version"`
	PreviousBundle Bundle          `yaml:"previous_bundle"`
}

// Refresh gets current host state and returns it as it is.
// func (hs HostState) Refresh(ctx context.Context, hst host.Host) (HostState, error) {
// 	typeNameStateMap, err := GetTypeNameResourceStateMap(ctx, hst, hs.PreviousBundle.Resources())
// 	if err != nil {
// 		return HostState{}, err
// 	}

// 	newBundle := Bundle{}
// 	for _, resources := range hs.PreviousBundle {
// 		newResources := Resources{}
// 		for _, resource := range resources {
// 			resourceState, ok := typeNameStateMap[resource.TypeName]
// 			if !ok {
// 				panic(fmt.Sprintf("missing ResourceState: %s", resource))
// 			}
// 			newResources = append(newResources, NewResource(
// 				resource.TypeName, resourceState.State, resourceState.State == nil),
// 			)
// 		}
// 		newBundle = append(newBundle, newResources)
// 	}
// 	return NewHostState(newBundle), nil
// }

func NewHostState(previousBundle Bundle) HostState {
	return HostState{
		Version:        version.GetVersion(),
		PreviousBundle: previousBundle,
	}
}

type ResourceState struct {
	State State
	Diffs []diffmatchpatch.Diff
	Clean bool
}

type TypeNameResourceStateMap map[TypeName]ResourceState

// GetMergeableManageableResourcesResourcesStateMapMap gets current state for all given resources,
// which must be MergeableManageableResources, and return it as TypeNameStateMap.
func GetMergeableManageableResourcesTypeNameResourceStateMap(
	ctx context.Context, hst host.Host, resources Resources,
) (TypeNameResourceStateMap, error) {
	logger := log.GetLogger(ctx)

	if len(resources) == 0 {
		return TypeNameResourceStateMap{}, nil
	}

	var mergeableManageableResources MergeableManageableResources

	names := []Name{}
	for _, resource := range resources {
		if !resource.IsMergeableManageableResources() {
			panic(fmt.Errorf("is not MergeableManageableResources: %s", resource))
		}
		if mergeableManageableResources == nil {
			mergeableManageableResources = resource.MustMergeableManageableResources()
		}
		names = append(names, resource.MustName())
	}

	nameStateMap, err := mergeableManageableResources.GetStates(ctx, hst, names)
	if err != nil {
		return nil, err
	}

	typeNameResourceStateMap := TypeNameResourceStateMap{}
	for _, resource := range resources {
		resourceState := ResourceState{}

		currentState, ok := nameStateMap[resource.MustName()]
		if !ok {
			panic(fmt.Errorf(
				"resource %s did not return state for %s", resource.MustType(), resource.MustName()),
			)
		}
		resourceState.State = currentState

		resourceState.Diffs = Diff(currentState, resource.State)

		if DiffsHasChanges(resourceState.Diffs) {
			logger.Infof("%s %s", ActionApply.Emoji(), resource)
			resourceState.Clean = false
		} else {
			logger.Infof("%s %s", ActionOk.Emoji(), resource)
			resourceState.Clean = true
		}

		typeNameResourceStateMap[resource.TypeName] = resourceState
	}

	return typeNameResourceStateMap, nil
}

type TypeNameStateMap map[TypeName]State

// GetTypeNameStateMap gets current state for all given TypeName.
func GetTypeNameStateMap(
	ctx context.Context, hst host.Host, typeNames []TypeName,
) (TypeNameStateMap, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🔎 Reading host state")
	nestedCtx := log.IndentLogger(ctx)

	// Separate individual from mergeable
	individuallyManageableResourcesTypeNames := []TypeName{}
	mergeableManageableResourcesTypeNameMap := map[Type][]Name{}
	for _, typeName := range typeNames {
		if typeName.IsIndividuallyManageableResource() {
			individuallyManageableResourcesTypeNames = append(
				individuallyManageableResourcesTypeNames, typeName,
			)
		} else if typeName.IsMergeableManageableResources() {
			mergeableManageableResourcesTypeNameMap[typeName.Type()] = append(
				mergeableManageableResourcesTypeNameMap[typeName.Type()], typeName.Name(),
			)
		} else {
			panic(fmt.Sprintf("unknow resource interface: %s", typeName))
		}
	}

	// TypeNameStateMap
	typeNameStateMap := TypeNameStateMap{}

	// Get state for individual
	for _, typeName := range individuallyManageableResourcesTypeNames {
		state, err := typeName.MustIndividuallyManageableResource().GetState(
			nestedCtx, hst, typeName.Name(),
		)
		if err != nil {
			return nil, err
		}
		typeNameStateMap[typeName] = state
	}

	// Get state for mergeable
	for tpe, names := range mergeableManageableResourcesTypeNameMap {
		nameStateMap, err := tpe.MustMergeableManageableResources().GetStates(nestedCtx, hst, names)
		if err != nil {
			return nil, err
		}
		for name, state := range nameStateMap {
			typeName, err := NewTypeName(tpe, name)
			if err != nil {
				panic(fmt.Sprintf("failed to create new TypeName %s %s: %s", tpe, name, err))
			}
			typeNameStateMap[typeName] = state
		}
	}

	return typeNameStateMap, nil
}
