package resource

import (
	"context"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/state"
)

// Instance holds parameters for a resource instance.
type Instance struct {
	// Name is a globally unique identifier for the resource.
	Name string `yaml:"name"`
	// Parameters holds resource specific parameters.
	// It must be marshallable by gopkg.in/yaml.v3.
	Parameters interface{} `yaml:"parameters"`
}

// Resource manages state.
type Resource interface {
	// Merge informs whether all resources from the same type are to be merged
	// together.
	// When true, all instances of the resource are joined as a single
	// instances slice with length equal to the number of resources for that
	// host.
	// When false, each instance of the resource is passed individually with
	// a instances slice of size 1.
	Merge() bool

	// Reads current resource state without any side effects.
	ReadState(
		ctx context.Context,
		host host.Host,
		instances []Instance,
	) (state.ResourceState, error)

	// Apply confiugres the resource at host to given instances state.
	// Must be idempotent.
	Apply(
		ctx context.Context,
		host host.Host,
		instances []Instance,
	) error

	// Destroy a configured resource at given host.
	// Must be idempotent.
	Destroy(
		ctx context.Context,
		host host.Host,
		instances []Instance,
	) error
}
