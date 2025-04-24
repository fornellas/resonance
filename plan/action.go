package plan

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	blueprintPkg "github.com/fornellas/resonance/blueprint"
	"github.com/fornellas/resonance/diff"
	"github.com/fornellas/resonance/host/types"
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
		var resourceAction *ResourceDiff
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
	for i, resourceDiff := range a.ResourceDiffs {
		actionStrs[i] = resourceDiff.String()
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
func (a *Action) Apply(ctx context.Context, host types.Host) error {
	ctx, logger := log.MustWithGroupAttrs(ctx, "🚀 Action: Apply", "resources", a.String())
	diffStr := a.DiffString()
	if len(diffStr) > 0 {
		ctx, logger = log.MustWithAttrs(ctx, "diff", diffStr)
		logger.Info("Applying changes")
	} else {
		logger.Debug("Nothing to do")
	}

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
