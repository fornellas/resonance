package state

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

type Host struct {
	Host host.Host
	Path string
}

func (h Host) Save(ctx context.Context, bytes []byte) error {
	logger := log.GetLogger(ctx)

	if err := h.Host.Mkdir(ctx, filepath.Dir(h.Path), 0700); err != nil {
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
	}

	if err := h.Host.WriteFile(ctx, h.Path, bytes, 0600); err != nil {
		return err
	}

	if err := h.Host.Chown(ctx, h.Path, 0, 0); err != nil {
		if !errors.Is(err, fs.ErrPermission) {
			return err
		}
		logger.Warnf(
			"failed to change ownership to root, be mindful it may contain sensitive information: %s",
			err,
		)
	}
	return nil
}

func (h Host) Load(ctx context.Context) (*[]byte, error) {
	logger := log.GetLogger(ctx)
	bytes, err := h.Host.ReadFile(ctx, h.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			logger.Debugf("File does not exist: %s", h.Path)
			return nil, nil
		}
		return nil, err
	}
	return &bytes, nil
}

func (h Host) String() string {
	return fmt.Sprintf("%s:%s", h.Host, h.Path)
}

// NewHost creates a new Host instance with Path set as a function of the root directory
// and the host.
func NewHost(root string, hst host.Host) Host {
	return Host{
		Host: hst,
		Path: filepath.Join(root, fmt.Sprintf("%s.yaml", hst.String())),
	}
}
