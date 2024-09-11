package discover

import (
	"context"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/resources"
)

// Interface for a package (eg: dpkg, rpm etc)
type Package interface {
	// The name of the package
	Name() string
	// Add symlink as broken
	AddBrokenSymLink(path string)
	// Add file inferred to be owned by package
	AddInferredOwnedPath(path string)
	// Return Resources that comply with the package.
	// group, if non empty, refers to how to group these resoures (eg: source package name).
	// issues contains a list of issues (eg: broken links, checksum failure, etc)
	GetResources(
		ctx context.Context, host types.Host,
	) (group string, resosurces resources.Resources, issues []string, err error)
}

// DB interface for packages (eg: apt, yum etc)
type PackageDb[P Package] interface {
	// Find packages that own given path
	FindOwnerPackages(path string) []P

	// Get package with given name
	GetPackage(name string) []P
}
