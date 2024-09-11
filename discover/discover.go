package discover

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/proc"
)

type Options struct {
	// Ignore files that match the patterns. See https://pkg.go.dev/path/filepath#Match.
	IgnorePatterns []string
	// Ignore files on these filesystems.
	IgnoreFsTypes []string
}

type Discover struct {
	Options Options
}

func NewDiscover(
	ctx context.Context,
	options Options,
) (*Discover, error) {
	return &Discover{
		Options: options,
	}, nil
}

//gocyclo:ignore
func (d *Discover) Run(ctx context.Context, host types.Host) error {
	mounts, err := proc.LoadMounts(ctx, host)
	if err != nil {
		return err
	}

	excludeFstypeMap := map[string]bool{}
	for _, path := range d.Options.IgnoreFsTypes {
		excludeFstypeMap[path] = true
	}

	ignorePatterns := d.Options.IgnorePatterns

	for _, mount := range mounts {
		if _, ok := excludeFstypeMap[mount.FSType]; !ok {
			continue
		}
		path := filepath.Clean(mount.MountPoint)
		ignorePatterns = append(ignorePatterns, path+"/*")
	}

	root, err := LoadRoot(ctx, host, ignorePatterns)
	if err != nil {
		return err
	}

	dpkgDb, err := LoadAptDb(ctx, host)
	if err != nil {
		return err
	}

	fileOwnership := NewOwnership(root, dpkgDb)
	fileOwnership.Compile(ctx, host)
	if err := fileOwnership.Report(ctx, host, os.Stdout); err != nil {
		return err
	}

	// TODO snap

	// TODO flatpack

	return nil
}
