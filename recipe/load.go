package recipe

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadFile reads and parses a single recipe file.
func LoadFile(ctx context.Context, path string) (*Recipe, error) {
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return Parse(ctx, yamlBytes, path)
}

// findYamlFiles resolves paths (each a file or a directory) into a sorted,
// deduplicated list of recipe file paths. A path naming a file is included
// as given, regardless of extension. A path naming a directory is walked
// recursively, including only files with a ".yaml" extension.
func findYamlFiles(paths ...string) ([]string, error) {
	seen := map[string]bool{}
	var files []string

	add := func(path string) error {
		abs, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if seen[abs] {
			return nil
		}
		seen[abs] = true
		files = append(files, path)
		return nil
	}

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			if err := add(path); err != nil {
				return nil, err
			}
			continue
		}

		err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || filepath.Ext(p) != ".yaml" {
				return nil
			}
			return add(p)
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Strings(files)

	return files, nil
}

// LoadPaths reads and merges the recipes found under paths (each a file or a
// directory).
func LoadPaths(ctx context.Context, paths ...string) (*Recipe, error) {
	yamlPaths, err := findYamlFiles(paths...)
	if err != nil {
		return nil, err
	}
	if len(yamlPaths) == 0 {
		return nil, fmt.Errorf("no .yaml recipe files found at %s", strings.Join(paths, ", "))
	}

	mergedRecipe := &Recipe{}

	for _, yamlPath := range yamlPaths {
		recipe, err := LoadFile(ctx, yamlPath)
		if err != nil {
			return nil, err
		}

		if err := mergedRecipe.Merge(recipe); err != nil {
			return nil, err
		}
	}

	return mergedRecipe, nil
}
