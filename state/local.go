package state

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/resource"
)

type Local struct {
	PersistantState
	Path string
}

func (l Local) Load(ctx context.Context) (resource.StateData, error) {
	f, err := os.Open(l.Path)
	if err != nil {
		return resource.StateData{}, fmt.Errorf("failed to load state from %s: %w", l.Path, err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	stateData := resource.StateData{}

	for {
		docStateData := resource.StateData{}
		if err := decoder.Decode(&docStateData); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return resource.StateData{}, fmt.Errorf("failed to load state: %s: %w", l.Path, err)
		}
		stateData.Merge(docStateData)
	}

	return stateData, nil
}
func (l Local) Save(ctx context.Context, stateData resource.StateData) error {
	return errors.New("TODO Local.Save")
}
