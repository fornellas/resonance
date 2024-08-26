package plan

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

func TestPlan(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	type testCase struct {
		name              string
		targetResources   resourcesPkg.Resources
		lastResources     resourcesPkg.Resources
		originalResources resourcesPkg.Resources
		expectedPlan      string
	}

	// These test cases go over various combinations of the resource existing
	// or not existing on the target / last blueprints, their states and
	// the original state.
	// Expectations are a function of the planned actions and the diff generated.
	testCases := []testCase{
		{
			name: "target=exists,last=absent,original=absent",
			targetResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "target",
				},
			},
			lastResources: resourcesPkg.Resources{},
			originalResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:   "/foo",
					Absent: true,
				},
			},
			expectedPlan: `File:🔧/foo
  path: /foo
  -absent: true
  +content: target`,
		},
		{
			name: "target!=original,last=absent,original=exists",
			targetResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "target",
				},
			},
			lastResources: resourcesPkg.Resources{},
			originalResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "original",
				},
			},
			expectedPlan: `File:🔄/foo
  path: /foo
  -content: original
  +content: target`,
		},
		{
			name: "target=original,last=absent,original=exists",
			targetResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "original",
				},
			},
			lastResources: resourcesPkg.Resources{},
			originalResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "original",
				},
			},
			expectedPlan: `File:✅/foo`,
		},
		{
			name:            "target=absent,last=exists,original=exists",
			targetResources: resourcesPkg.Resources{},
			lastResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "last",
				},
			},
			originalResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "original",
				},
			},
			expectedPlan: `File:🔄/foo
  path: /foo
  -content: last
  +content: original`,
		},
		{
			name: "target=last,last=exists,original=exists",
			targetResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "last",
				},
			},
			lastResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "last",
				},
			},
			originalResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "original",
				},
			},
			expectedPlan: `File:✅/foo`,
		},
		{
			name: "target=!last,last=exists,original=exists",
			targetResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "target",
				},
			},
			lastResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "last",
				},
			},
			originalResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "original",
				},
			},
			expectedPlan: `File:🔄/foo
  path: /foo
  -content: last
  +content: target`,
		},
		{
			name: "target=absent,last=exists,original=exists",
			targetResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:   "/foo",
					Absent: true,
				},
			},
			lastResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "last",
				},
			},
			originalResources: resourcesPkg.Resources{
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "original",
				},
			},
			expectedPlan: `File:🗑/foo
  path: /foo
  -content: last
  +absent: true`,
		},
		{
			name: "GroupResource+SingleResource",
			targetResources: resourcesPkg.Resources{
				&resourcesPkg.APTPackage{
					Package: "barPkg",
					Version: "3.5.target",
				},
				&resourcesPkg.File{
					Path:    "/baz",
					Content: "target",
				},
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "target",
				},
			},
			lastResources: resourcesPkg.Resources{
				&resourcesPkg.APTPackage{
					Package: "barPkg",
					Version: "3.4.last",
				},
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "last",
				},
				&resourcesPkg.APTPackage{
					Package: "fooPkg",
					Version: "1.2.last",
				},
				&resourcesPkg.File{
					Path:    "/bar",
					Content: "last",
				},
			},
			originalResources: resourcesPkg.Resources{
				&resourcesPkg.APTPackage{
					Package: "barPkg",
					Version: "3.4.original",
				},
				&resourcesPkg.File{
					Path:    "/foo",
					Content: "original",
				},
				&resourcesPkg.APTPackage{
					Package: "fooPkg",
					Version: "1.2.original",
				},
				&resourcesPkg.File{
					Path:    "/bar",
					Content: "original",
				},
				&resourcesPkg.File{
					Path:    "/baz",
					Content: "original",
				},
			},
			expectedPlan: `APTPackages:🔄barPkg,🔄fooPkg
  barPkg:
    package: barPkg
    -version: 3.4.last
    +version: 3.5.target
  fooPkg:
    package: fooPkg
    -version: 1.2.last
    +version: 1.2.original
File:🔄/baz
  path: /baz
  -content: original
  +content: target
File:🔄/foo
  path: /foo
  -content: last
  +content: target
File:🔄/bar
  path: /bar
  -content: last
  +content: original`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetBlueprint, err := blueprintPkg.NewBlueprintFromResources(ctx, tc.targetResources)
			require.NoError(t, err)

			lastBlueprint, err := blueprintPkg.NewBlueprintFromResources(ctx, tc.lastResources)
			require.NoError(t, err)

			originalResourceMap := resourcesPkg.NewResourceMap(tc.originalResources)

			plan, err := NewPlan(
				ctx,
				targetBlueprint, lastBlueprint,
				func(ctx context.Context, resource resourcesPkg.Resource) (resourcesPkg.Resource, error) {
					return originalResourceMap.GetResourceWithSameTypeId(resource), nil
				},
			)
			require.NoError(t, err)

			require.GreaterOrEqual(t, len(plan), len(targetBlueprint.Steps))

			var buff bytes.Buffer
			for _, action := range plan {
				fmt.Fprintf(&buff, "%s\n", action.DetailedString())
				idsWithDiff := []string{}
				for _, resourceDiff := range action.ResourceDiffs {
					if len(resourceDiff.Chunks) > 0 {
						idsWithDiff = append(idsWithDiff, resourceDiff.Id)
					}
				}
				applyIds := []string{}
				for _, applyResource := range action.ApplyResources {
					applyIds = append(applyIds, resourcesPkg.GetResourceId(applyResource))
				}
				require.Equal(t, idsWithDiff, applyIds)
			}
			planStr := strings.TrimSuffix(buff.String(), "\n")
			require.Equal(t, tc.expectedPlan, planStr)
		})
	}
}
