package plan

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	hostPkg "github.com/fornellas/resonance/host"
	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
	"github.com/fornellas/resonance/internal/diff"
	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

// ResourceDiff holds the diff for applying each Action resource.
type ResourceDiff struct {
	// An emoji representing the action
	Emoji rune
	// The resource Id
	Id string
	// The diff to apply the resource.
	Chunks diff.Chunks
}

func (r *ResourceDiff) String() string {
	return fmt.Sprintf("%c%s", r.Emoji, r.Id)
}

// NewResourceDiff creates a new ResourceDiff
func NewResourceDiff(emoji rune, planResource resourcesPkg.Resource, chunks diff.Chunks) *ResourceDiff {
	return &ResourceDiff{
		Emoji:  emoji,
		Id:     resourcesPkg.GetResourceId(planResource),
		Chunks: chunks,
	}
}

// Action holds actions required to apply, calculated from a Step.
type Action struct {
	// A string with the resource type (eg: File, APTPackage)
	ResourceType string
	// The calculated diff for each Step resource when applying it.
	ResourceDiffs []*ResourceDiff
	// The resources that needs to be applied for the Step (eg: if no changes required for the
	// resource, it won't be here)
	ApplyResources resourcesPkg.Resources
}

// NewAction creates a new Action for a given Step, calculating required changes as a function
// of the before (apply) state of the step resources.
func NewAction(step *blueprintPkg.Step, beforeResourceMap resourcesPkg.ResourceMap) *Action {
	action := &Action{
		ResourceType:   step.Type(),
		ResourceDiffs:  make([]*ResourceDiff, len(step.Resources())),
		ApplyResources: resourcesPkg.Resources{},
	}
	for j, planResource := range step.Resources() {
		beforeResource := beforeResourceMap.GetResourceWithSameTypeId(planResource)
		if beforeResource == nil {
			panic("bug: before resource not found")
		}
		var resourceAction *ResourceDiff = nil
		if resourcesPkg.Satisfies(beforeResource, planResource) {
			resourceAction = NewResourceDiff('✅', planResource, nil)
		} else {
			action.ApplyResources = append(action.ApplyResources, planResource)
			var emoji rune
			if resourcesPkg.GetResourceAbsent(beforeResource) {
				emoji = '🔧'
			} else {
				if resourcesPkg.GetResourceAbsent(planResource) {
					emoji = '🗑'
				} else {
					emoji = '🔄'
				}
			}
			resourceAction = NewResourceDiff(emoji, planResource, diff.DiffAsYaml(
				beforeResource, planResource,
			))
		}
		action.ResourceDiffs[j] = resourceAction
	}
	slices.SortFunc(action.ResourceDiffs, func(a, b *ResourceDiff) int {
		return strings.Compare(a.Id, b.Id)
	})
	slices.SortFunc(action.ApplyResources, func(a, b resourcesPkg.Resource) int {
		return strings.Compare(resourcesPkg.GetResourceId(a), resourcesPkg.GetResourceId(b))
	})
	return action
}

// String returns a single-line representation of the action.
func (a *Action) String() string {
	actionStrs := make([]string, len(a.ResourceDiffs))
	for i, resourceDiffs := range a.ResourceDiffs {
		actionStrs[i] = resourceDiffs.String()
	}
	return fmt.Sprintf("%s:%s", a.ResourceType, strings.Join(actionStrs, ","))
}

// DiffString returns details on the diff required to apply this action.
func (a *Action) DiffString() string {
	var buff bytes.Buffer

	if len(a.ResourceDiffs) == 1 {
		resourceDiffs := a.ResourceDiffs[0]
		if len(resourceDiffs.Chunks) > 0 {
			fmt.Fprintf(&buff, "%s", resourceDiffs.Chunks.String())
		}
	} else {
		for _, resourceDiffs := range a.ResourceDiffs {
			if len(resourceDiffs.Chunks) > 0 {
				fmt.Fprintf(&buff, "%s:\n", resourceDiffs.Id)
				fmt.Fprintf(&buff, "  %s\n",
					strings.Join(
						strings.Split(
							strings.TrimSuffix(resourceDiffs.Chunks.String(), "\n"),
							"\n",
						),
						"\n  ",
					),
				)
			}
		}
	}

	return strings.TrimSuffix(buff.String(), "\n")
}

// DetailedString returns a multi-line string, fully describing the action and its diff.
func (a *Action) DetailedString() string {
	diffStr := a.DiffString()
	if len(diffStr) > 0 {
		return strings.TrimSuffix(
			fmt.Sprintf("%s\n  %s\n",
				a,
				strings.Join(
					strings.Split(
						strings.TrimSuffix(diffStr, "\n"),
						"\n",
					),
					"\n  ",
				),
			),
			"\n",
		)
	} else {
		return a.String()
	}
}

// Apply commits all required changes for action to given Host.
func (a *Action) Apply(ctx context.Context, host hostPkg.Host) error {
	args := []any{}
	diffStr := a.DiffString()
	if len(diffStr) > 0 {
		args = append(args, []any{"diff", diffStr}...)
	}
	ctx, _ = log.MustContextLoggerSection(ctx, a.String(), args...)

	if len(a.ApplyResources) == 0 {
		return nil
	}

	isGroupResource := resourcesPkg.IsGroupResource(
		resourcesPkg.GetResourceTypeName(a.ApplyResources[0]),
	)

	if isGroupResource {
		groupResource := resourcesPkg.GetGroupResourceByTypeName(a.ResourceType)
		if groupResource == nil {
			panic("bug: bad GroupResource")
		}
		if err := groupResource.Apply(ctx, host, a.ApplyResources); err != nil {
			return err
		}
	} else {
		if len(a.ApplyResources) != 1 {
			panic("bug: can't have more than one SingleResource")
		}
		singleResource, ok := a.ApplyResources[0].(resourcesPkg.SingleResource)
		if !ok {
			panic("bug: is not SingleResource")
		}
		if err := singleResource.Apply(ctx, host); err != nil {
			return err
		}
	}

	return nil
}

// Plan holds all actions required to apply changes to a host.
// This enables evaluating all changes before they are applied.
type Plan []*Action

// NewPlan crafts a new plan as a function of: the delined target state, the last state (how the
// host is at the moment) and the original state of any managed resource.
func NewPlan(
	ctx context.Context,
	targetBlueprint, lastBlueprint *blueprintPkg.Blueprint,
	loadOriginalResource func(ctx context.Context, resource resourcesPkg.Resource) (resourcesPkg.Resource, error),
) (Plan, error) {
	targetResources := targetBlueprint.Resources()
	planResources := make(resourcesPkg.Resources, len(targetResources))
	beforeResources := make(resourcesPkg.Resources, len(targetResources))

	// Add every target resource to plan
	for i, targetResource := range targetResources {
		planResources[i] = targetResource
		// Before resource is a function of last resource existing or not
		lastResource := lastBlueprint.GetResourceWithSameTypeId(targetResource)
		if lastResource == nil {
			originalResource, err := loadOriginalResource(ctx, targetResource)
			if err != nil {
				return nil, err
			}
			beforeResources[i] = originalResource
		} else {
			beforeResources[i] = lastResource
		}
	}

	// each last resource that is not on target must be restored to original
	type ToOriginal struct {
		lastIdx  int
		resource resourcesPkg.Resource
	}
	toOriginalSlice := []ToOriginal{}
	lastResources := lastBlueprint.Resources()
	for i, lastResource := range lastResources {
		if targetBlueprint.HasResourceWithSameTypeId(lastResource) {
			continue
		}
		originalResource, err := loadOriginalResource(ctx, lastResource)
		if err != nil {
			return nil, err
		}
		toOriginalSlice = append(toOriginalSlice, ToOriginal{
			lastIdx:  i,
			resource: originalResource,
		})
		beforeResources = append(beforeResources, lastResource)
	}

	// merge original resources with plan resources, respecting the original order
	for _, toOriginal := range toOriginalSlice {
		idx := -1
		for i := toOriginal.lastIdx + 1; i < len(lastResources) && idx < 0; i++ {
			lastResourceTypeId := resourcesPkg.GetResourceTypeId(lastResources[i])
			for j, planResource := range planResources {
				planResourceTypeId := resourcesPkg.GetResourceTypeId(planResource)
				if planResourceTypeId == lastResourceTypeId {
					idx = j
					break
				}
			}
		}
		if idx < 0 {
			idx = len(planResources)
		}
		planResources = append(
			planResources[:idx],
			append(resourcesPkg.Resources{toOriginal.resource}, planResources[idx:]...)...,
		)
	}

	// Calculate plan actions
	planBlueprint, err := blueprintPkg.NewBlueprintFromResources(ctx, planResources)
	if err != nil {
		return nil, err
	}
	beforeResourceMap := resourcesPkg.NewResourceMap(beforeResources)
	plan := make(Plan, len(planBlueprint.Steps))
	for i, step := range planBlueprint.Steps {
		plan[i] = NewAction(step, beforeResourceMap)
	}
	return plan, nil
}

// Apply commits all required changes for plan to given Host.
func (p Plan) Apply(ctx context.Context, host hostPkg.Host) error {
	ctx, _ = log.MustContextLoggerSection(ctx, "⚙️ Applying")

	for _, action := range p {
		if err := action.Apply(ctx, host); err != nil {
			return err
		}
	}

	return nil
}
