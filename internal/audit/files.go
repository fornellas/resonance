package audit

import (
	"context"
	"path/filepath"

	hostPkg "github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

type FilesDb map[string]bool

func findFiles(
	ctx context.Context,
	host hostPkg.Host,
	path string,
	filesDb FilesDb,
	excludePathsMap map[string][]string,
) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Listting", "path", path)

	dirEnts, err := host.ReadDir(ctx, path)
	if err != nil {
		return err
	}
	for _, dirEnt := range dirEnts {
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
			if err := findFiles(ctx, host, fullPath, filesDb, excludePathsMap); err != nil {
				return err
			}
		} else {
			filesDb[fullPath] = true
		}
	}

	return nil
}

func NewFilesDb(
	ctx context.Context,
	host hostPkg.Host,
	excludePathsMap map[string][]string,
) (FilesDb, error) {
	ctx, _ = log.MustContextLoggerSection(ctx, "Finding all files")

	filesDb := FilesDb{}

	if err := findFiles(ctx, host, "/", filesDb, excludePathsMap); err != nil {
		return nil, err
	}

	return filesDb, nil
}
