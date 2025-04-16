package blueprint

import (
	"bytes"
	"context"
	"fmt"

	"github.com/fornellas/resonance/diff"
	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

// Blueprint holds a full desired state for a host.
type Blueprint struct {
	Steps       Steps
	resourceMap resourcesPkg.ResourceMap
}

// NewBlueprintFromResources creates a new Blueprint from given Resources, merging GroupResource
// of the same type in the same step, while respecting the declared order.
func NewBlueprintFromResources(ctx context.Context, resources resourcesPkg.Resources) (*Blueprint, error) {
	steps, err := NewSteps(resources)
	if err != nil {
		return nil, err
	}

	blueprint := &Blueprint{
		Steps: steps,
	}

	return blueprint, nil
}

func (b *Blueprint) getresourceMap() resourcesPkg.ResourceMap {
	if b.resourceMap != nil {
		return b.resourceMap
	}
	b.resourceMap = resourcesPkg.NewResourceMap(b.Resources())
	return b.resourceMap
}

func (b *Blueprint) String() string {
	var buff bytes.Buffer
	for _, step := range b.Steps {
		fmt.Fprintf(&buff, "%s\n", step)
	}
	return buff.String()
}

// Resolve the state with information that may be required from the host for all Resources.
func (b *Blueprint) Resolve(ctx context.Context, hst types.Host) error {
	ctx, _ = log.WithGroup(ctx, "⚙️ Resolving")
	for _, step := range b.Steps {
		if err := step.Resolve(ctx, hst); err != nil {
			return err
		}
	}
	return nil
}

// Load returns a copy of the Blueprint, with all resource states loaded from given Host.
func (b *Blueprint) Load(ctx context.Context, hst types.Host) (*Blueprint, error) {
	newBlueprint := &Blueprint{
		Steps: make(Steps, len(b.Steps)),
	}
	for i, step := range b.Steps {
		newStep, err := step.Load(ctx, hst)
		if err != nil {
			return nil, err
		}
		newBlueprint.Steps[i] = newStep
	}
	return newBlueprint, nil
}

// Returns all resources from all steps, ordered.
func (b *Blueprint) Resources() resourcesPkg.Resources {
	resources := resourcesPkg.Resources{}
	if b == nil {
		return resources
	}
	for _, step := range b.Steps {
		resources = append(resources, step.Resources()...)
	}
	return resources
}

func (b *Blueprint) GetResourceWithSameTypeId(resource resourcesPkg.Resource) resourcesPkg.Resource {
	return b.getresourceMap().GetResourceWithSameTypeId(resource)
}

func (b *Blueprint) HasResourceWithSameTypeId(resource resourcesPkg.Resource) bool {
	return b.getresourceMap().GetResourceWithSameTypeId(resource) != nil
}

// Satisfies returns whether the Blueprint satisfies otherBlueprint.
// Eg: if the Blueprint defines a package with a name and a specific version, but
// otherBlueprint specifies a package with the same name, but with any version, then
// the Blueprint satisfies otherBlueprint.
func (b *Blueprint) Satisfies(otherBlueprint *Blueprint) diff.Chunks {
	chunks := diff.Chunks{}

	visitedResources := map[resourcesPkg.Resource]bool{}

	for _, resource := range b.Resources() {
		visitedResources[resource] = true
		otherResource := otherBlueprint.GetResourceWithSameTypeId(resource)
		if otherResource == nil || !resourcesPkg.Satisfies(resource, otherResource) {
			chunks = append(chunks, diff.DiffAsYaml(otherResource, resource)...)
		}
	}

	for _, otherResource := range otherBlueprint.Resources() {
		if _, ok := visitedResources[otherResource]; ok {
			continue
		}
		resource := b.GetResourceWithSameTypeId(otherResource)
		if resource == nil || !resourcesPkg.Satisfies(resource, otherResource) {
			chunks = append(chunks, diff.DiffAsYaml(otherResource, resource)...)
		}
	}

	return chunks
}
