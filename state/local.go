package state

import (
	"context"
	"errors"
	"io/fs"
	"os"
)

type Local struct {
	Path string
}

func (l Local) Save(ctx context.Context, bytes []byte) error {
	if err := os.WriteFile(l.Path, bytes, 0600); err != nil {
		return err
	}
	return nil
}

func (l Local) Load(ctx context.Context) (*[]byte, error) {
	bytes, err := os.ReadFile(l.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return &bytes, nil
}

func (l Local) String() string {
	return l.Path
}
