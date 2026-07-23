package recipe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// testParseValid iterates over all YAML fixtures under dir, asserting that
// each one parses successfully.
func testParseValid(t *testing.T, dir string) {
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(dir, entry.Name())
			yamlBytes, err := os.ReadFile(path)
			require.NoError(t, err)

			r, err := Parse(t.Context(), yamlBytes, path)
			require.NoError(t, err)
			require.NotNil(t, r)
		})
	}
}

// testParseInvalid iterates over all YAML fixtures under dir, asserting that
// each one fails to parse with an error containing the matching entry in
// expectedErrors (keyed by file name). Fails if dir's contents and
// expectedErrors' keys don't match exactly, so fixtures and expectations
// can't silently drift apart.
func testParseInvalid(t *testing.T, dir string, expectedErrors map[string]string) {
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	fixtureNames := make([]string, len(entries))
	for i, entry := range entries {
		fixtureNames[i] = entry.Name()
	}
	expectedNames := make([]string, 0, len(expectedErrors))
	for name := range expectedErrors {
		expectedNames = append(expectedNames, name)
	}
	require.ElementsMatch(t, fixtureNames, expectedNames, "%s fixtures and expectedErrors table must match", dir)

	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(dir, entry.Name())
			yamlBytes, err := os.ReadFile(path)
			require.NoError(t, err)

			r, err := Parse(t.Context(), yamlBytes, path)
			require.Error(t, err)
			require.Nil(t, r)
			require.ErrorContains(t, err, expectedErrors[entry.Name()])
		})
	}
}
