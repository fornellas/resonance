package store

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
)

// Implementation of Store that persists Blueprints at a Host at Path.
type HostStore struct {
	Host      types.Host
	logPath   string
	statePath string
}

// NewHostStore creates a new HostStore for given Host.
func NewHostStore(host types.Host, path string) *HostStore {
	basePath := filepath.Join(path, "state", "v1")
	return &HostStore{
		Host:      host,
		logPath:   filepath.Join(path, "logs"),
		statePath: basePath,
	}
}

func (s *HostStore) deleteOldLogs(ctx context.Context) error {
	dirEntResultCh, cancel := s.Host.ReadDir(ctx, s.logPath)
	defer cancel()

	var names []string
	for dirEntResult := range dirEntResultCh {
		if dirEntResult.Error != nil {
			if errors.Is(dirEntResult.Error, os.ErrNotExist) {
				return nil
			}
			return dirEntResult.Error
		}
		dirEnt := dirEntResult.DirEnt
		if filepath.Ext(dirEnt.Name) == ".gz" {
			names = append(names, dirEnt.Name)
		}
	}

	if len(names) <= 10 {
		return nil
	}

	sort.Strings(names)

	namesToDelete := names[:len(names)-10]

	for _, name := range namesToDelete {
		path := filepath.Join(s.logPath, name)
		if err := s.Host.Remove(ctx, path); err != nil {
			return err
		}
	}

	return nil
}

func (s *HostStore) GetLogWriterCloser(ctx context.Context, name string) (io.WriteCloser, error) {
	if err := s.deleteOldLogs(ctx); err != nil {
		return nil, err
	}

	if err := lib.MkdirAll(ctx, s.Host, s.logPath, 0700); err != nil {
		return nil, err
	}

	gzipWriter, err := gzip.NewWriterLevel(
		&lib.HostFileWriter{
			Context: ctx,
			Host:    s.Host,
			Path: filepath.Join(
				s.logPath,
				fmt.Sprintf("%s.%s.gz", time.Now().UTC().Format("20060102150405"), name),
			),
		},
		gzip.BestSpeed,
	)
	if err != nil {
		return nil, err
	}
	if err := gzipWriter.Flush(); err != nil {
		return nil, err
	}
	return gzipWriter, nil
}
