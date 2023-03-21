package resource

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"

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
	RollbackBundle *Bundle         `yaml:"rollback_bundle"`
}

func (hs HostState) String() string {
	bytes, err := yaml.Marshal(&hs)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

// Refresh gets current host state and returns it as it is.
func (hs HostState) Refresh(
	ctx context.Context, typeNameStateMap TypeNameStateMap,
) (HostState, error) {
	newBundle := Bundle{}
	for _, resources := range hs.PreviousBundle {
		newResources := Resources{}
		for _, resource := range resources {
			currentState, ok := typeNameStateMap[resource.TypeName]
			if !ok {
				panic(fmt.Sprintf("missing ResourceState: %s", resource))
			}
			newResources = append(newResources, NewResource(
				resource.TypeName, currentState, currentState == nil),
			)
		}
		newBundle = append(newBundle, newResources)
	}
	return NewHostState(newBundle, nil), nil
}

// IsClean checks whether host state is clean.
func (hs HostState) IsClean(
	ctx context.Context,
	hst host.Host,
	typeNameStateMap TypeNameStateMap,
) (string, error) {
	previousBundleIsClean, err := hs.PreviousBundle.IsClean(ctx, hst, typeNameStateMap)
	if err != nil {
		return "", err
	}
	if !previousBundleIsClean {
		return "Previous host state is not clean: this often means external agents altered the host state after a previous action. Try the 'refresh' or 'restore' commands.", nil
	}

	if hs.RollbackBundle != nil {
		return "A previous action did not fully complete and a rollback is pending, try the 'rollback' command.", nil
	}
	return "", nil
}

func NewHostState(previousBundle Bundle, rollbackBundle *Bundle) HostState {
	return HostState{
		Version:        version.GetVersion(),
		PreviousBundle: previousBundle,
		RollbackBundle: rollbackBundle,
	}
}

type TypeNameStateMap map[TypeName]State

// GetTypeNameStateMap gets current state for all given TypeName.
func GetTypeNameStateMap(
	ctx context.Context, hst host.Host, typeNames []TypeName,
) (TypeNameStateMap, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🔎 Reading host state")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

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
		nestedLogger.Infof("%s", typeName)
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
