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

func (l Local) Load(ctx context.Context) (resource.HostState, error) {
	hostState := resource.HostState{}

	f, err := os.Open(l.Path)
	if err != nil {
		return hostState, fmt.Errorf("failed to load state: %w", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	for {
		docState := resource.HostState{}
		if err := decoder.Decode(&docState); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return hostState, fmt.Errorf("failed to load state: %s: %w", l.Path, err)
		}
		hostState.Merge(docState)
	}

	return hostState, nil
}
func (l Local) Save(ctx context.Context, hostState resource.HostState) error {
	return errors.New("TODO Local.Save")
}
