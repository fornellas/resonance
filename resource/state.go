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
	currentResourcesState ResourcesState,
) error {
	logger := log.GetLogger(ctx)
	logger.Info("🕵️ Checking host state")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	fail := false

	for _, resource := range hs.Resources {
		isClean, ok := currentResourcesState.TypeNameCleanMap[resource.TypeName]
		if !ok {
			panic(fmt.Sprintf("state missing from StateMap: %s", resource))
		}
		if !isClean {
			nestedLogger.Errorf("%s state is not clean", resource)
			fail = true
		}
	}

	if fail {
		return errors.New("state is dirty: this means external changes happened to the host that should be addressed before proceeding. Check refresh / restore commands and / or fix the changes manually")
	}

	return nil
}

// Refresh updates each resource from HostState.Resources to the current state and return
// the new HostState
func (hs HostState) Refresh(ctx context.Context, hst host.Host) (HostState, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🔁 Refreshing state")
	nestedCtx := log.IndentLogger(ctx)

	newHostState := NewHostState(Resources{})

	for _, resource := range hs.Resources {
		currentState, err := resource.ManageableResource().GetState(
			nestedCtx, hst, resource.TypeName.Name(),
		)
		if err != nil {
			return HostState{}, err
		}

		newHostState.Resources = append(newHostState.Resources, NewResource(
			resource.TypeName, currentState, resource.Destroy,
		))
	}

	return newHostState, nil
}

func NewHostState(resources Resources) HostState {
	return HostState{
		Version:   version.GetVersion(),
		Resources: resources,
	}
}

type ResourcesState struct {
	TypeNameStateMap map[TypeName]State
	TypeNameCleanMap map[TypeName]bool
	TypeNameDiffsMap map[TypeName][]diffmatchpatch.Diff
}

func NewResourcesState(ctx context.Context, hst host.Host, resources Resources) (ResourcesState, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🔎 Reading host state")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	resourcesState := ResourcesState{
		TypeNameStateMap: map[TypeName]State{},
		TypeNameCleanMap: map[TypeName]bool{},
		TypeNameDiffsMap: map[TypeName][]diffmatchpatch.Diff{},
	}
	for _, resource := range resources {
		currentState, err := resource.ManageableResource().GetState(nestedCtx, hst, resource.TypeName.Name())
		if err != nil {
			return ResourcesState{}, err
		}
		resourcesState.TypeNameStateMap[resource.TypeName] = currentState

		diffs, err := resource.ManageableResource().DiffStates(nestedCtx, hst, resource.State, currentState)
		if err != nil {
			return ResourcesState{}, err
		}

		resourcesState.TypeNameDiffsMap[resource.TypeName] = diffs

		if DiffsHasChanges(diffs) {
			nestedLogger.Infof("%s %s", ActionApply.Emoji(), resource)
			resourcesState.TypeNameCleanMap[resource.TypeName] = false
		} else {
			nestedLogger.Infof("%s %s", ActionOk.Emoji(), resource)
			resourcesState.TypeNameCleanMap[resource.TypeName] = true
		}

	}
	return resourcesState, nil
}
