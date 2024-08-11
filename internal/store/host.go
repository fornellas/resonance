package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
	"github.com/fornellas/resonance/log"
)

// Implementation of Store that persists Blueprints at a Host at Path.
type HostStore struct {
	Host host.Host
	Path string
}

// NewHostStore creates a new HostStore for given Host.
func NewHostStore(hst host.Host, path string) *HostStore {
	return &HostStore{
		Host: hst,
		Path: path,
	}
}

func (s *HostStore) getYamlPath() string {
	return filepath.Join(s.Path, "blueprint.yaml")
}

func (s *HostStore) GetLastBlueprint(ctx context.Context) (blueprintPkg.Blueprint, error) {
	logger := log.MustLogger(ctx)
	logger.Info("📂 Loading last stored Bueprint")
	blueprintBytes, err := s.Host.ReadFile(ctx, s.getYamlPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	blueprint := blueprintPkg.Blueprint{}
	err = yaml.Unmarshal(blueprintBytes, &blueprint)
	if err != nil {
		return nil, err
	}

	return blueprint, nil
}

func (s *HostStore) Save(ctx context.Context, blueprint blueprintPkg.Blueprint) error {
	logger := log.MustLogger(ctx)
	logger.Info("💾 Saving Bueprint to host")

	blueprintBytes, err := yaml.Marshal(blueprint)
	if err != nil {
		panic(fmt.Sprintf("bug: failed to serialize blueprint: %s", err.Error()))
	}

	dir := filepath.Dir(s.getYamlPath())

	if err := os.MkdirAll(dir, fs.FileMode(0700)); err != nil {
		return err
	}

	return s.Host.WriteFile(ctx, s.getYamlPath(), blueprintBytes, os.FileMode(0600))
}
