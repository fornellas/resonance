package state

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

type Local struct {
	Path string
}

func (l Local) Save(ctx context.Context, bytes []byte) error {
	if err := os.MkdirAll(filepath.Dir(l.Path), 0700); err != nil {
		return err
	}

	if err := os.WriteFile(l.Path, bytes, 0600); err != nil {
		return err
	}
	return nil
}

func (l Local) Load(ctx context.Context) (*[]byte, error) {
	logger := log.GetLogger(ctx)
	bytes, err := os.ReadFile(l.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			logger.Debugf("File does not exist: %s", l.Path)
			return nil, nil
		}
		return nil, err
	}
	return &bytes, nil
}

func (l Local) String() string {
	return l.Path
}

// NewLocal creates a new Local instance with Path set as a function of the root directory
// and the host.
func NewLocal(root string, hst host.Host) Local {
	return Local{
		Path: filepath.Join(root, fmt.Sprintf("%s.yaml", hst.String())),
	}
}
