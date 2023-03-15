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
	resourcesStateMap, err := NewResourcesStateMap(ctx, hst, hs.Resources)
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

func NewResourceState(ctx context.Context, hst host.Host, resource Resource) (ResourceState, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🔎 Reading host state")

	resourcesState := ResourceState{}

	currentState, err := resource.ManageableResource().GetState(ctx, hst, resource.TypeName.Name())
	if err != nil {
		return ResourceState{}, err
	}
	resourcesState.State = currentState

	diffs, err := resource.ManageableResource().DiffStates(ctx, hst, resource.State, currentState)
	if err != nil {
		return ResourceState{}, err
	}

	resourcesState.Diffs = diffs

	if DiffsHasChanges(diffs) {
		logger.Infof("%s %s", ActionApply.Emoji(), resource)
		resourcesState.Clean = false
	} else {
		logger.Infof("%s %s", ActionOk.Emoji(), resource)
		resourcesState.Clean = true
	}

	return resourcesState, nil
}

type ResourcesStateMap map[TypeName]ResourceState

func NewResourcesStateMap(ctx context.Context, hst host.Host, resources Resources) (ResourcesStateMap, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🔎 Reading host state")
	nestedCtx := log.IndentLogger(ctx)

	resourcesStateMap := ResourcesStateMap{}
	for _, resource := range resources {
		resourcesState, err := NewResourceState(nestedCtx, hst, resource)
		if err != nil {
			return ResourcesStateMap{}, err
		}
		resourcesStateMap[resource.TypeName] = resourcesState
	}
	return resourcesStateMap, nil
}
