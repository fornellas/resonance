package recipe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindYamlFiles(t *testing.T) {
	t.Run("directory recursion, .yaml filter", func(t *testing.T) {
		files, err := findYamlFiles("testdata/load/dir")
		require.NoError(t, err)
		require.Equal(t, []string{"testdata/load/dir/a.yaml", "testdata/load/dir/sub/b.yaml"}, files)
	})

	t.Run("explicit file, any extension", func(t *testing.T) {
		files, err := findYamlFiles("testdata/load/dir/notes.txt")
		require.NoError(t, err)
		require.Equal(t, []string{"testdata/load/dir/notes.txt"}, files)
	})

	t.Run("same file reachable directly and via a directory is deduped", func(t *testing.T) {
		files, err := findYamlFiles("testdata/load/dir", "testdata/load/dir/a.yaml")
		require.NoError(t, err)
		require.Equal(t, []string{"testdata/load/dir/a.yaml", "testdata/load/dir/sub/b.yaml"}, files)
	})

	t.Run("no such path", func(t *testing.T) {
		_, err := findYamlFiles("testdata/load/does-not-exist")
		require.Error(t, err)
	})
}

func TestLoadFile(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r, err := LoadFile(t.Context(), "testdata/load/multi-a.yaml")
		require.NoError(t, err)
		require.Len(t, r.Files, 1)
	})

	t.Run("no such file", func(t *testing.T) {
		_, err := LoadFile(t.Context(), "testdata/load/does-not-exist.yaml")
		require.Error(t, err)
	})
}

func TestLoadPaths(t *testing.T) {
	t.Run("merges multiple files given as args", func(t *testing.T) {
		r, err := LoadPaths(t.Context(), "testdata/load/multi-a.yaml", "testdata/load/multi-b.yaml")
		require.NoError(t, err)
		require.Len(t, r.Files, 2)
	})

	t.Run("merges a directory", func(t *testing.T) {
		r, err := LoadPaths(t.Context(), "testdata/load/dir")
		require.NoError(t, err)
		require.Len(t, r.Files, 2)
	})

	t.Run("file declared in two different files names both locations", func(t *testing.T) {
		_, err := LoadPaths(t.Context(), "testdata/load/dup-a.yaml", "testdata/load/dup-b.yaml")
		require.Error(t, err)
		require.ErrorContains(t, err, `file "/etc/dup" already declared at`)
		require.ErrorContains(t, err, "testdata/load/dup-a.yaml:2:9")
		require.ErrorContains(t, err, "testdata/load/dup-b.yaml:2:9")
	})

	t.Run("package declared in two different files names both locations", func(t *testing.T) {
		_, err := LoadPaths(t.Context(), "testdata/load/dup-pkg-a.yaml", "testdata/load/dup-pkg-b.yaml")
		require.Error(t, err)
		require.ErrorContains(t, err, `apt_package "nginx" already declared at`)
		require.ErrorContains(t, err, "testdata/load/dup-pkg-a.yaml:2:12")
		require.ErrorContains(t, err, "testdata/load/dup-pkg-b.yaml:2:12")
	})

	t.Run("no yaml files found", func(t *testing.T) {
		_, err := LoadPaths(t.Context(), "testdata/load/no-yaml")
		require.Error(t, err)
	})
}
