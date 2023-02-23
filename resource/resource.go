package resource

import (
	"context"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/state"
)

type Parameter struct {
	// Name is a globally unique identifier for the resource
	Name string
	// Data holds resource specific parameters.
	// It must be marshallable by gopkg.in/yaml.v3.
	Data interface{}
}

// Resource manages state.
type Resource interface {
	// Merge informs whether all resources from the same type are to be merged
	// together.
	// When true, all instances of the resource are joined as a single
	// parameters slice with length equal to the number of resources for that
	// host.
	// When false, each instance of the resource is passed individually with
	// a parameters slice of size 1.
	Merge() bool

	// Reads current resource state
	ReadState(
		ctx context.Context,
		host host.Host,
		parameters []Parameter,
	) (state.ResourceState, error)

	// Apply confiugres the resource to given host based on its parameters.
	// Must be idempotent.
	Apply(
		ctx context.Context,
		host host.Host,
		parameters []Parameter,
	) error

	// Destroy a configured resource at given host.
	// Must be idempotent.
	Destroy(
		ctx context.Context,
		host host.Host,
		parameters []Parameter,
	) error
}

// func ApplyResource(
// 	ctx context.Context,
// 	host host.Host,
// 	resource Resource,
// 	parameters []Parameter,
// ) error {
// 	savedState, err := state.Load(name)
// 	if err != nil {
// 		return err
// 	}

// 	if savedState != nil {
// 		preState, err := resource.ReadState(ctx, host, name, parameters)
// 		if err != nil {
// 			return err
// 		}
// 		if reflect.DeepEqual(*savedState, preState) {
// 			return nil
// 		}
// 	}

// 	if err := resource.Apply(ctx, host, name, parameters); err != nil {
// 		return err
// 	}

// 	applyState, err := resource.ReadState(ctx, host, name, parameters)
// 	if err != nil {
// 		return err
// 	}
// 	if err := state.Save(applyState); err != nil {
// 		return err
// 	}

// 	return nil
// }
