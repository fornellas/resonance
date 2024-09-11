package discover

import (
	"context"
	"fmt"
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

// Compiles list of all ignore patterns for the givenhost
func (o *Options) GetIngorePatterns(ctx context.Context, host types.Host) ([]string, error) {
	mounts, err := proc.LoadMounts(ctx, host)
	if err != nil {
		return nil, err
	}

	excludeFstypeMap := map[string]bool{}
	for _, path := range o.IgnoreFsTypes {
		excludeFstypeMap[path] = true
	}

	ignorePatterns := o.IgnorePatterns

	for _, mount := range mounts {
		if _, ok := excludeFstypeMap[mount.FSType]; !ok {
			continue
		}
		path := filepath.Clean(mount.MountPoint)
		ignorePatterns = append(ignorePatterns, path+"/*")
	}

	return ignorePatterns, nil
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

func (d *Discover) prepareResourcesPath(resourcesPath string) error {
	info, err := os.Stat(resourcesPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(resourcesPath, 0700); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if !info.IsDir() {
			return fmt.Errorf("%#v exists but is not a directory", resourcesPath)
		}
		entries, err := os.ReadDir(resourcesPath)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf("directory %#v is not empty", resourcesPath)
		}
	}
	return nil
}

func (d *Discover) Run(ctx context.Context, host types.Host, resourcesPath string) error {
	if err := d.prepareResourcesPath(resourcesPath); err != nil {
		return err
	}

	ignorePatterns, err := d.Options.GetIngorePatterns(ctx, host)
	if err != nil {
		return err
	}

	root, err := LoadRoot(ctx, host, ignorePatterns)
	if err != nil {
		return err
	}

	dpkgDb, err := LoadAptDb(ctx, host)
	if err != nil {
		return err
	}

	ownership := NewOwnership(root, dpkgDb)
	ownership.Compile(ctx, host)
	if err := ownership.CompileResources(ctx, host, resourcesPath); err != nil {
		return err
	}

	// TODO rpm / yum

	// TODO snap

	// TODO flatpack

	// TODO systemd

	// TODO debian alternatives

	return nil
}
