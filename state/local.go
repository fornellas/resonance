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
	Path string
}

func (l Local) Load(ctx context.Context) ([]resource.ResourceDefinition, error) {
	resourceDefinitions := []resource.ResourceDefinition{}

	f, err := os.Open(l.Path)
	if err != nil {
		return resourceDefinitions, fmt.Errorf("failed to load state: %w", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	hasMultipleDocuments := false
	for {
		if err := decoder.Decode(&resourceDefinitions); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return resourceDefinitions, fmt.Errorf("failed to load: %s: %w", l.Path, err)
		}
		if hasMultipleDocuments {
			return resourceDefinitions, fmt.Errorf("expected to have a single document: %s", l.Path)
		}
		hasMultipleDocuments = true
	}

	return resourceDefinitions, nil
}
func (l Local) Save(ctx context.Context, hostState []resource.ResourceDefinition) error {
	return errors.New("TODO Local.Save")
}
