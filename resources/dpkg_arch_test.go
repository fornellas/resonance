package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDpkgArch(t *testing.T) {
	t.Run("Satisfies()", func(t *testing.T) {
		a := &DpkgArch{ForeignArchitectures: []string{"i386", "arm64"}}
		b := &DpkgArch{ForeignArchitectures: []string{"i386"}}
		require.True(t, a.Satisfies(b))
		require.False(t, b.Satisfies(a))
		require.True(t, a.Satisfies(&DpkgArch{ForeignArchitectures: []string{}}))
		require.True(t, (&DpkgArch{ForeignArchitectures: []string{}}).Satisfies(&DpkgArch{ForeignArchitectures: []string{}}))
		require.False(t, (&DpkgArch{ForeignArchitectures: []string{}}).Satisfies(a))
	})
	t.Run("Validate()", func(t *testing.T) {
		require.NoError(t, (&DpkgArch{ForeignArchitectures: []string{"i386", "amd64"}}).Validate())
		require.Error(t, (&DpkgArch{ForeignArchitectures: []string{"!"}}).Validate())
		require.Error(t, (&DpkgArch{ForeignArchitectures: []string{"amd64", "bad!"}}).Validate())
	})
}
