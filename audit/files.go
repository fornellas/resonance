package audit

import (
	"context"
	"path/filepath"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

type FilesDb map[string]bool

func (f FilesDb) findFiles(
	ctx context.Context,
	host types.Host,
	path string,
	excludePathsMap map[string][]string,
) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Listting", "path", path)

	dirEntResultCh, cancel := host.ReadDir(ctx, path)
	defer cancel()
	for dirEntResult := range dirEntResultCh {
		if dirEntResult.Error != nil {
			return dirEntResult.Error
		}
		dirEnt := &dirEntResult.DirEnt
		fullPath := filepath.Join(path, dirEnt.Name)
		if reason, ok := excludePathsMap[fullPath]; ok {
			logger.Debug("Skipping", "path", fullPath, "reason", reason)
			continue
		}
		if dirEnt.IsDirectory() {
			// FIXME
			if dirEnt.Name == "__pycache__" {
				continue
			}
			if dirEnt.IsSymbolicLink() {
				continue
			}
			if err := f.findFiles(ctx, host, fullPath, excludePathsMap); err != nil {
				return err
			}
		} else {
			f[fullPath] = true
		}
	}

	return nil
}

func NewFilesDb(
	ctx context.Context,
	host types.Host,
	excludePathsMap map[string][]string,
) (FilesDb, error) {
	ctx, _ = log.MustContextLoggerSection(ctx, "Finding all files")

	filesDb := FilesDb{}

	if err := findFiles(ctx, host, "/", filesDb, excludePathsMap); err != nil {
		return nil, err
	}

	return filesDb, nil
}
