package blueprint

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

func TestNewBlueprintFromResources(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	aptFoo := &resourcesPkg.APTPackage{
		Package: "foo",
	}
	fileFoo := &resourcesPkg.File{
		Path: "/foo",
	}
	aptBar := &resourcesPkg.APTPackage{
		Package: "bar",
	}
	fileBar := &resourcesPkg.File{
		Path: "/bar",
	}

	blueprint, err := NewBlueprintFromResources(ctx, resourcesPkg.Resources{
		aptFoo, fileFoo, aptBar, fileBar,
	})
	require.NoError(t, err)

	stepStrs := make([]string, len(blueprint.Steps))
	for i, step := range blueprint.Steps {
		stepStrs[i] = step.String()
	}

	require.Equal(t, []string{
		"APTPackages:bar,foo",
		"File:/foo",
		"File:/bar",
	}, stepStrs)

	require.Equal(t, resourcesPkg.Resources{
		aptBar, aptFoo, fileFoo, fileBar,
	}, blueprint.Resources())

	require.Equal(t, aptFoo, blueprint.GetResourceWithSameTypeId(aptFoo))
	require.Nil(t, blueprint.GetResourceWithSameTypeId(&resourcesPkg.File{Path: "/new"}))

	require.True(t, blueprint.HasResourceWithSameTypeId(aptFoo))
	require.False(t, blueprint.HasResourceWithSameTypeId(&resourcesPkg.File{Path: "/new"}))
}
