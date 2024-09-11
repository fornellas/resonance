package audit

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/proc"
)

type Audit struct {
	host            types.Host
	excludePathsMap map[string][]string
}

func NewAudit(
	ctx context.Context,
	host types.Host,
	excludePaths []string,
	excludeFsTypes []string,
) (*Audit, error) {
	audit := &Audit{
		host:            host,
		excludePathsMap: map[string][]string{},
	}

	// TODO fail if `uid -u` != 0 (not root)

	for _, path := range excludePaths {
		audit.excludePathsMap[path] = append(audit.excludePathsMap[path], "excluded path")
	}

	excludeFstypeMap := map[string]bool{}
	for _, path := range excludeFsTypes {
		excludeFstypeMap[path] = true
	}

	mounts, err := proc.NewMounts(ctx, host)
	if err != nil {
		return nil, err
	}
	for _, mount := range mounts {
		if _, ok := excludeFstypeMap[mount.FSType]; ok {
			path := mount.MountPoint
			audit.excludePathsMap[path] = append(audit.excludePathsMap[path], "excluded fstype")
		}
	}

	return audit, nil
}

func (a *Audit) Run(ctx context.Context) error {
	filesDb, err := NewFilesDb(
		ctx,
		a.host,
		a.excludePathsMap,
	)
	if err != nil {
		return err
	}

	dpkgDb, err := NewDpkgDb(ctx, a.host)
	if err != nil {
		return err
	}

	// fmt.Printf("dpkgDb\n")
	// fmt.Printf("  DpkgPackages\n")
	// for _, dpkgPackage := range dpkgDb.DpkgPackages {
	// 	fmt.Printf("    binary:Package: %s\n", dpkgPackage.BinaryPackage)
	// 	fmt.Printf("    Version: %s\n", dpkgPackage.Version)
	// 	fmt.Printf("    db-fsys:Files:\n")
	// 	for _, f := range dpkgPackage.DbFsysFiles {
	// 		fmt.Printf("      %s\n", f)
	// 	}
	// 	fmt.Printf("    source:Package: %s\n", dpkgPackage.SourcePackage)
	// 	fmt.Printf("    Conffiles:\n")
	// 	for _, f := range dpkgPackage.Conffiles {
	// 		fmt.Printf("      %s\n", f)
	// 	}
	// 	fmt.Printf("\n")
	// }
	// fmt.Printf("  PathToDpkgPackagesMap\n")
	// fmt.Printf("    %d\n", len(dpkgDb.PathToDpkgPackagesMap))
	// files := []string{}
	// for file := range dpkgDb.PathToDpkgPackagesMap {
	// 	files = append(files, file)
	// }
	// sort.Strings(files)
	// for _, file := range files {
	// 	fmt.Printf("    %#v\n", file)
	// 	for _, dpkgPackage := range dpkgDb.PathToDpkgPackagesMap[file] {
	// 		fmt.Printf("      %#v\n", dpkgPackage.BinaryPackage)
	// 	}
	// }

	// FIXME add exclusion for common paths, that by chance, may be owned by a single package
	// Eg: /usr/share, /var/lib, /usr/lib etc
	excludePackageOwnerPaths := map[string]bool{
		"/etc":                true,
		"/etc/pam.d":          true,
		"/usr":                true,
		"/usr/bin":            true,
		"/usr/include":        true,
		"/usr/lib":            true,
		"/usr/libexec":        true,
		"/usr/lib/tmpfiles.d": true,
		"/usr/sbin":           true,
		"/usr/share":          true,
		"/usr/share/doc":      true,
		"/usr/share/man":      true,
		// FIXME /usr/share/man should be enough
		"/usr/share/man/man1": true,
		"/usr/share/man/man2": true,
		"/usr/share/man/man3": true,
		"/usr/share/man/man4": true,
		"/usr/share/man/man5": true,
		"/usr/share/man/man6": true,
		"/usr/share/man/man7": true,
		"/usr/share/man/man8": true,
	}

	ownerMap := map[string][]string{}
	for path := range filesDb {
		owner := "(orphan)"
		if _, ok := dpkgDb.PathToDpkgPackagesMap[path]; ok {
			continue
			// if len(dpkgPackages) == 1 {
			// 	owner = dpkgPackages[0].SourcePackage
			// } else {
			// 	owner = "(multiple packages)"
			// }
		} else {
			parentPath := path
			for {
				parentPath = filepath.Dir(parentPath)
				if parentPath == "/" {
					break
				}
				if _, ok := excludePackageOwnerPaths[parentPath]; ok {
					break
				}
				if dpkgPackages, ok := dpkgDb.PathToDpkgPackagesMap[parentPath]; ok {
					switch len(dpkgPackages) {
					case 0:
						panic("bug: empty packages list")
					case 1:
						owner = dpkgPackages[0].SourcePackage
						// fmt.Printf("Found: path %#v, via %#v is owned by %#v\n", path, parentPath, owner)
					default:
					}
					break
				}

			}
		}
		ownerMap[owner] = append(ownerMap[owner], path)
	}

	fmt.Printf("RESULTS\n")
	owners := []string{}
	for owner := range ownerMap {
		owners = append(owners, owner)
	}
	sort.Strings(owners)
	for _, owner := range owners {
		paths := ownerMap[owner]
		sort.Strings(paths)
		fmt.Printf("  %s:\n", owner)
		for _, path := range paths {
			fmt.Printf("    %s\n", path)
		}
	}

	// TODO orphan files
	// - if now owned by any package
	// - try to match with each source package path

	// TODO verify all packages
	// - if corrupted, just log it
	// - if config files, consider for automation

	return nil
}
