package resource

import (
	"context"
	"fmt"

	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/version"
)

// State is a Type specific interface for defining resource state as configured by users.
type State interface {
	// Validate whether the parameters are OK.
	Validate() error
}

// HostState holds the state for a host
type HostState struct {
	// Version of the binary used to put the host in this state.
	Version        version.Version `yaml:"version"`
	PreviousBundle Bundle          `yaml:"previous_bundle"`
}

func (hs HostState) String() string {
	bytes, err := yaml.Marshal(&hs)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

// IsClean whether current host state matches HostState.
func (hs HostState) IsClean(
	ctx context.Context,
	hst host.Host,
	typeNameResourceStateMap TypeNameResourceStateMap,
) bool {
	logger := log.GetLogger(ctx)
	logger.Info("🕵️ Checking host state")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	clean := true

	for _, resource := range hs.PreviousBundle.Resources() {
		resourcesState, ok := typeNameResourceStateMap[resource.TypeName]
		if !ok {
			panic(fmt.Sprintf("state missing from StateMap: %s", resource))
		}
		if !resourcesState.Clean {
			nestedLogger.Errorf("%s state is not clean", resource)
			clean = false
		}
	}

	return clean
}

// Refresh gets current host state and returns it as it is.
func (hs HostState) Refresh(ctx context.Context, hst host.Host) (HostState, error) {
	typeNameStateMap, err := GetTypeNameStateMap(ctx, hst, hs.PreviousBundle.Resources())
	if err != nil {
		return HostState{}, err
	}

	newBundle := Bundle{}
	for _, resources := range hs.PreviousBundle {
		newResources := Resources{}
		for _, resource := range resources {
			resourceState, ok := typeNameStateMap[resource.TypeName]
			if !ok {
				panic(fmt.Sprintf("missing ResourceState: %s", resource))
			}
			newResources = append(newResources, NewResource(
				resource.TypeName, resourceState.State, resourceState.State == nil),
			)
		}
		newBundle = append(newBundle, newResources)
	}
	return NewHostState(newBundle), nil
}

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

// GetIndividuallyManageableResourceResourceState gets current state for all given resources,
// which must be IndividuallyManageableResource, and return it as ResourceState.
func GetIndividuallyManageableResourceResourceState(
	ctx context.Context, hst host.Host, resource Resource,
) (ResourceState, error) {
	logger := log.GetLogger(ctx)

	resourceState := ResourceState{}

	individuallyManageableResource := resource.MustIndividuallyManageableResource()

	currentState, err := individuallyManageableResource.GetState(ctx, hst, resource.TypeName.Name())
	if err != nil {
		return ResourceState{}, err
	}
	resourceState.State = currentState

	if resource.State != nil && currentState != nil {
		diffs, err := individuallyManageableResource.DiffStates(ctx, hst, resource.State, currentState)
		if err != nil {
			return ResourceState{}, err
		}
		resourceState.Diffs = diffs
	} else {
		resourceState.Diffs = Diff(currentState, resource.State)
	}

	if DiffsHasChanges(resourceState.Diffs) {
		logger.Infof("%s %s", ActionApply.Emoji(), resource)
		resourceState.Clean = false
	} else {
		logger.Infof("%s %s", ActionOk.Emoji(), resource)
		resourceState.Clean = true
	}

	return resourceState, nil
}

// GetMergeableManageableResourcesResourcesStateMapMap gets current state for all given resources,
// which must be MergeableManageableResources, and return it as TypeNameStateMap.
func GetMergeableManageableResourcesTypeNameStateMap(
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

		if resource.State != nil && currentState != nil {
			diffs, err := mergeableManageableResources.DiffStates(ctx, hst, resource.State, currentState)
			if err != nil {
				return nil, err
			}
			resourceState.Diffs = diffs
		} else {
			resourceState.Diffs = Diff(currentState, resource.State)
		}

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

// GetTypeNameStateMap gets current state for all given resources and return
// it as TypeNameStateMap.
func GetTypeNameStateMap(
	ctx context.Context, hst host.Host, resources Resources,
) (TypeNameResourceStateMap, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🔎 Reading host state")
	nestedCtx := log.IndentLogger(ctx)

	individuallyManageableResources := []Resource{}
	typeMergeableManageableResourcesMap := map[Type][]Resource{}
	for _, resource := range resources {
		if resource.IsMergeableManageableResources() {
			typeMergeableManageableResourcesMap[resource.MustType()] = append(
				typeMergeableManageableResourcesMap[resource.MustType()], resource,
			)
		} else {
			individuallyManageableResources = append(individuallyManageableResources, resource)
		}
	}

	typeNameResourceStateMap := TypeNameResourceStateMap{}

	for _, individuallyManageableResource := range individuallyManageableResources {
		resourcesState, err := GetIndividuallyManageableResourceResourceState(
			nestedCtx, hst, individuallyManageableResource,
		)
		if err != nil {
			return TypeNameResourceStateMap{}, err
		}
		typeNameResourceStateMap[individuallyManageableResource.TypeName] = resourcesState
	}

	for _, mergeableManageableResources := range typeMergeableManageableResourcesMap {
		mergeableManageableResourcesTypeNameStateMap, err := GetMergeableManageableResourcesTypeNameStateMap(
			nestedCtx, hst, mergeableManageableResources,
		)
		if err != nil {
			return TypeNameResourceStateMap{}, err
		}

		for typeName, resourceState := range mergeableManageableResourcesTypeNameStateMap {
			typeNameResourceStateMap[typeName] = resourceState
		}
	}

	return typeNameResourceStateMap, nil
}
