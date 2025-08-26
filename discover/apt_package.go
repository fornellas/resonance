package discover

import (
	"context"
	"fmt"
	"sync"

	"github.com/fornellas/resonance/host/types"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

type AptPackage struct {
	pkg          string
	architecture string
	version      string
	// all package files
	dbFsysFiles   []string
	sourcePackage string
	// config files
	conffiles []string
	// dbFsysFiles that are broken symlinks
	brokenSymLinks []string
	// files which were inferred to be owned by this package
	inferredOwnedPaths []string
	// dbFsysFiles which are missing
	missingPaths []string
	// dbFsysFiles which digest check failed
	digestCheckFailedPaths []string
	// apt marked as manual
	manual bool
	// apt marked as hold
	hold  bool
	mutex sync.Mutex
}

func (a *AptPackage) Name() string {
	return a.pkg + ":" + a.architecture
}

func (a *AptPackage) AddBrokenSymLink(path string) {
	a.mutex.Lock()
	a.brokenSymLinks = append(a.brokenSymLinks, path)
	a.mutex.Unlock()
}

func (a *AptPackage) AddInferredOwnedPath(path string) {
	a.mutex.Lock()
	a.inferredOwnedPaths = append(a.inferredOwnedPaths, path)
	a.mutex.Unlock()
}

func (a *AptPackage) AddMissingPath(path string) {
	a.mutex.Lock()
	a.missingPaths = append(a.missingPaths, path)
	a.mutex.Unlock()
}

func (a *AptPackage) AddDigestCheckFailedPath(path string) {
	a.mutex.Lock()
	a.digestCheckFailedPaths = append(a.digestCheckFailedPaths, path)
	a.mutex.Unlock()
}

func (a *AptPackage) MarkManual() {
	a.mutex.Lock()
	a.manual = true
	a.mutex.Unlock()
}

func (a *AptPackage) MarkHold() {
	a.mutex.Lock()
	a.hold = true
	a.mutex.Unlock()
}

func (a *AptPackage) String() string {
	return a.Name() + "-" + a.version
}

func (a *AptPackage) CompileResources(
	ctx context.Context, host types.Host,
) (string, resourcesPkg.Resources, []string, error) {
	group := a.sourcePackage

	resources := resourcesPkg.Resources{}
	if a.manual || len(a.inferredOwnedPaths) > 0 {
		var version string
		if a.hold {
			version = a.version
		}
		resources = append(resources, &resourcesPkg.APTPackage{
			Package: a.pkg,
			Version: version,
			// FIXME merge all APTPackage with same name
			Architectures: []string{
				a.architecture,
			},
		})
		for _, path := range a.inferredOwnedPaths {
			file := &resourcesPkg.File{
				Path: path,
			}
			if err := file.Load(ctx, host); err != nil {
				return "", nil, nil, err
			}
			resources = append(resources, file)
		}
	}

	issues := []string{}
	for _, path := range a.brokenSymLinks {
		issues = append(issues, fmt.Sprintf("broken symlink: %s", path))
	}
	for _, path := range a.missingPaths {
		issues = append(issues, fmt.Sprintf("missing: %s", path))
	}
	for _, path := range a.digestCheckFailedPaths {
		issues = append(issues, fmt.Sprintf("digest check failed: %s", path))
	}

	return group, resources, issues, nil
}
