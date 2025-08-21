package state

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"

	"github.com/fornellas/resonance/resources"
)

// Load parses HCL configuration files and returns a State object with all declared resources
// Duplicate resources are merged when possible, or return conflicts errors with source locations
func Load(filePaths []string) (*State, error) {
	parser := hclparse.NewParser()
	var allDiagnostics hcl.Diagnostics
	var states []*State

	// Parse each file
	for _, filePath := range filePaths {
		var file *hcl.File
		var diags hcl.Diagnostics

		// Determine file type by extension
		ext := filepath.Ext(filePath)
		switch ext {
		case ".hcl":
			file, diags = parser.ParseHCLFile(filePath)
		case ".json":
			file, diags = parser.ParseJSONFile(filePath)
		default:
			// Default to HCL format
			file, diags = parser.ParseHCLFile(filePath)
		}

		allDiagnostics = append(allDiagnostics, diags...)
		if file == nil {
			continue
		}

		// Decode the file content directly into State
		state := &State{}
		decodeDiags := gohcl.DecodeBody(file.Body, nil, state)
		allDiagnostics = append(allDiagnostics, decodeDiags...)

		states = append(states, state)
	}

	// Check for parsing errors
	if allDiagnostics.HasErrors() {
		return nil, fmt.Errorf("HCL parsing errors: %s", allDiagnostics.Error())
	}

	// Merge all states, handling duplicates
	mergedState := &State{
		Files:       []*resources.File{},
		APTPackages: []*resources.APTPackage{},
	}

	if err := mergeStates(mergedState, states); err != nil {
		return nil, err
	}

	return mergedState, nil
}

// mergeStates merges multiple states into one, handling duplicate resources
func mergeStates(target *State, states []*State) error {
	// Use maps to track resources by their key for merging
	filesByPath := make(map[string]*resources.File)
	aptPackagesByName := make(map[string]*resources.APTPackage)

	// Process all files
	for _, state := range states {
		for _, file := range state.Files {
			if existingFile, exists := filesByPath[file.Path]; exists {
				// Attempt to merge
				if err := existingFile.Merge(file); err != nil {
					return fmt.Errorf("failed to merge file %s: %w", file.Path, err)
				}
			} else {
				// Create a copy to avoid modifying the original
				fileCopy := *file
				filesByPath[file.Path] = &fileCopy
			}
		}
	}

	// Process all APT packages
	for _, state := range states {
		for _, aptPackage := range state.APTPackages {
			if existingPackage, exists := aptPackagesByName[aptPackage.Package]; exists {
				// Attempt to merge
				if err := existingPackage.Merge(aptPackage); err != nil {
					return fmt.Errorf("failed to merge APT package %s: %w", aptPackage.Package, err)
				}
			} else {
				// Create a copy to avoid modifying the original
				packageCopy := *aptPackage
				aptPackagesByName[aptPackage.Package] = &packageCopy
			}
		}
	}

	// Convert maps back to slices
	for _, file := range filesByPath {
		target.Files = append(target.Files, file)
	}

	for _, aptPackage := range aptPackagesByName {
		target.APTPackages = append(target.APTPackages, aptPackage)
	}

	return nil
}
