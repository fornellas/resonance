package resource

import (
	"context"

	"github.com/fornellas/resonance/host"
)

type ResourceParameters interface{}

// Instance holds parameters for a resource instance.
type Instance struct {
	// Name is a globally unique identifier for the resource.
	Name string `yaml:"name"`
	// Parameters holds resource specific parameters.
	// It must be marshallable by gopkg.in/yaml.v3.
	Parameters ResourceParameters `yaml:"parameters"`
}

// Resource manages state.
type Resource interface {
	// Merge informs whether all resources from the same type are to be merged
	// together when applying.
	// When true, Apply is called only once, with all instances.
	// When false, Apply is called one time for each instance.
	Merge() bool

	// Reads current resource state without any side effects.
	ReadState(
		ctx context.Context,
		host host.Host,
		instances []Instance,
	) (ResourceState, error)

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
