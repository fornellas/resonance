package resources

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fornellas/resonance/log"
)

type StateDefinition struct {
	File *FileState
	// AptPackage *AptPackageState
}

func (s *StateDefinition) State() (State, error) {
	states := []State{}
	if s.File != nil {
		states = append(states, s.File)
	}
	// if s.AptPackage != nil {
	// 	states = append(states, s.AptPackage)
	// }
	switch len(states) {
	case 0:
		return nil, fmt.Errorf("no state defined")
	case 1:
		return states[0], nil
	default:
		return nil, fmt.Errorf("only one state can be defined")
	}
}

type StatesDefinition []StateDefinition

func LoadStatesDefinitionFromDir(ctx context.Context, dir string) ([]*StatesDefinition, error) {
	ctx, logger := log.MustContextLoggerWithSection(ctx, "🗃️ Loading resources from directory", "dir", dir)

	stateDefinitions := []*StatesDefinition{}

	paths := []string{}
	if err := filepath.Walk(dir, func(path string, fileInfo fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fileInfo.IsDir() || !strings.HasSuffix(fileInfo.Name(), ".yaml") {
			logger.Debug("Skipping", "path", path)
			return nil
		}
		logger.Debug("Found resources file", "path", path)
		paths = append(paths, path)
		return nil
	}); err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no .yaml resource files found under %s", dir)
	}
	sort.Strings(paths)

	for _, path := range paths {
		fileResources, err := LoadFile(ctx, path)
		if err != nil {
			return nil, err
		}
		stateDefinitions = append(stateDefinitions, fileResources...)
	}

	if err := stateDefinitions.Validate(); err != nil {
		return stateDefinitions, err
	}

	return stateDefinitions, nil
}

func LoadStatesDefinitionFromPath(ctx context.Context, path string) ([]*StatesDefinition, error) {
	var stateDefinitions []*StatesDefinition

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() {
		stateDefinitions, err = LoadStatesDefinitionFromDir(ctx, path)
		if err != nil {
			return nil, err
		}
	} else {
		stateDefinitions, err = LoadStatesDefinitionFromFile(ctx, path)
		if err != nil {
			return nil, err
		}
	}

	// TODO ensure unique name for each state type
	return stateDefinitions, nil
}

func (s StatesDefinition) States() ([]State, error) {
	states := make([]State, len(s))
	for i, stateSchema := range s {
		var err error
		states[i], err = stateSchema.State()
		if err != nil {
			return nil, err
		}
	}
	return states, nil
}
