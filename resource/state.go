package resource

import (
	"context"
	"errors"
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
	Version   version.Version `yaml:"version"`
	Resources Resources       `yaml:"resources"`
}

func (hs HostState) String() string {
	bytes, err := yaml.Marshal(&hs)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

// Check whether current host state matches HostState.
func (hs HostState) Check(
	ctx context.Context,
	hst host.Host,
	currentResourcesStateMap ResourcesStateMap,
) error {
	logger := log.GetLogger(ctx)
	logger.Info("🕵️ Checking host state")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	fail := false

	for _, resource := range hs.Resources {
		resourcesState, ok := currentResourcesStateMap[resource.TypeName]
		if !ok {
			panic(fmt.Sprintf("state missing from StateMap: %s", resource))
		}
		if !resourcesState.Clean {
			nestedLogger.Errorf("%s state is not clean", resource)
			fail = true
		}
	}

	if fail {
		return errors.New("state is dirty: this means external changes happened to the host that should be addressed before proceeding. Check refresh / restore commands and / or fix the changes manually")
	}

	return nil
}

// Refresh gets current host state and returns it as it is.
func (hs HostState) Refresh(ctx context.Context, hst host.Host) (HostState, error) {
	resourcesStateMap, err := GetResourcesStateMap(ctx, hst, hs.Resources)
	if err != nil {
		return HostState{}, err
	}

	resources := Resources{}
	for _, savedResource := range hs.Resources {
		resourceState, ok := resourcesStateMap[savedResource.TypeName]
		if !ok {
			panic(fmt.Sprintf("missing ResourceState: %s", savedResource))
		}
		resources = append(resources, NewResource(
			savedResource.TypeName, resourceState.State, resourceState.State == nil),
		)
	}
	return NewHostState(resources), nil
}

func NewHostState(resources Resources) HostState {
	return HostState{
		Version:   version.GetVersion(),
		Resources: resources,
	}
}

type ResourceState struct {
	State State
	Diffs []diffmatchpatch.Diff
	Clean bool
}

type ResourcesStateMap map[TypeName]ResourceState

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

	diffs, err := individuallyManageableResource.DiffStates(ctx, hst, resource.State, currentState)
	if err != nil {
		return ResourceState{}, err
	}
	resourceState.Diffs = diffs

	if DiffsHasChanges(diffs) {
		logger.Infof("%s %s", ActionApply.Emoji(), resource)
		resourceState.Clean = false
	} else {
		logger.Infof("%s %s", ActionOk.Emoji(), resource)
		resourceState.Clean = true
	}

	return resourceState, nil
}

// GetMergeableManageableResourcesResourcesStateMapMap gets current state for all given resources,
// which must be MergeableManageableResources, and return it as ResourcesStateMap.
func GetMergeableManageableResourcesResourcesStateMapMap(
	ctx context.Context, hst host.Host, resources Resources,
) (ResourcesStateMap, error) {
	logger := log.GetLogger(ctx)

	if len(resources) == 0 {
		return ResourcesStateMap{}, nil
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

	resourcesStateMap := ResourcesStateMap{}
	for _, resource := range resources {
		resourceState := ResourceState{}

		currentState, ok := nameStateMap[resource.MustName()]
		if !ok {
			panic(fmt.Errorf(
				"resource %s did not return state for %s", resource.MustType(), resource.MustName()),
			)
		}
		resourceState.State = currentState

		diffs, err := mergeableManageableResources.DiffStates(ctx, hst, resource.State, currentState)
		if err != nil {
			return nil, err
		}
		resourceState.Diffs = diffs

		if DiffsHasChanges(diffs) {
			logger.Infof("%s %s", ActionApply.Emoji(), resource)
			resourceState.Clean = false
		} else {
			logger.Infof("%s %s", ActionOk.Emoji(), resource)
			resourceState.Clean = true
		}

		resourcesStateMap[resource.TypeName] = resourceState
	}

	return resourcesStateMap, nil
}

// GetResourcesStateMap gets current state for all given resources and return
// it as ResourcesStateMap.
func GetResourcesStateMap(ctx context.Context, hst host.Host, resources Resources) (ResourcesStateMap, error) {
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

	resourcesStateMap := ResourcesStateMap{}

	for _, individuallyManageableResource := range individuallyManageableResources {
		resourcesState, err := GetIndividuallyManageableResourceResourceState(
			nestedCtx, hst, individuallyManageableResource,
		)
		if err != nil {
			return ResourcesStateMap{}, err
		}
		resourcesStateMap[individuallyManageableResource.TypeName] = resourcesState
	}

	for _, mergeableManageableResources := range typeMergeableManageableResourcesMap {
		mergeableManageableResourcesResourcesStateMap, err := GetMergeableManageableResourcesResourcesStateMapMap(
			ctx, hst, mergeableManageableResources,
		)
		if err != nil {
			return ResourcesStateMap{}, err
		}

		for typeName, resourceState := range mergeableManageableResourcesResourcesStateMap {
			resourcesStateMap[typeName] = resourceState
		}
	}

	return resourcesStateMap, nil
}
