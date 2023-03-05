package state

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"
)

// PersistantState defines an interface for loading and saving HostState.
type PersistantState interface {
	// Saves given state data.
	Save(ctx context.Context, bytes []byte) error
	// Loads a previously saved state data. If no previous state exists, returns nil.
	Load(ctx context.Context) (*[]byte, error)
	String() string
}

// LoadHostState loads a ResourceBundle saved after it was applied to a host.
func LoadHostState(ctx context.Context, persistantState PersistantState) (resource.HostState, error) {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)

	var hostState resource.HostState

	logger.Infof("📂 Loading saved state from %s", persistantState)

	savedBytes, err := persistantState.Load(nestedCtx)
	if err != nil {
		return hostState, err
	}

	decoder := yaml.NewDecoder(bytes.NewReader(*savedBytes))
	decoder.KnownFields(true)
	hasMultipleDocuments := false
	for {
		if err := decoder.Decode(&hostState); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return hostState, fmt.Errorf("failed to load: %w", err)
		}
		if hasMultipleDocuments {
			return hostState, fmt.Errorf("saved state has multiple documents")
		}
		hasMultipleDocuments = true
	}

	return hostState, nil
}
