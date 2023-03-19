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

func SaveHostState(ctx context.Context, hostState resource.HostState, persistantState PersistantState) error {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)

	logger.Infof("💾 Saving new host state to %s", persistantState)

	bytes, err := yaml.Marshal(&hostState)
	if err != nil {
		panic(fmt.Errorf("failed to yaml.Marshal state: %w", err))
	}
	if err := persistantState.Save(nestedCtx, bytes); err != nil {
		return err
	}
	return nil
}

func LoadHostState(
	ctx context.Context, persistantState PersistantState,
) (*resource.HostState, error) {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	var hostState resource.HostState

	logger.Infof("📂 Loading saved host state from %s", persistantState)

	savedBytes, err := persistantState.Load(nestedCtx)
	if err != nil {
		return &hostState, err
	}
	if savedBytes == nil {
		nestedLogger.Info("No previously saved state")
		return nil, nil
	}

	decoder := yaml.NewDecoder(bytes.NewReader(*savedBytes))
	decoder.KnownFields(true)
	hasMultipleDocuments := false
	for {
		if err := decoder.Decode(&hostState); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return &hostState, fmt.Errorf("failed to load: %w", err)
		}
		if hasMultipleDocuments {
			return &hostState, fmt.Errorf("saved state has multiple documents")
		}
		hasMultipleDocuments = true
	}

	return &hostState, nil
}
