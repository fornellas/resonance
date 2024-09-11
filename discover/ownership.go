package discover

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"syscall"

	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

// Infers file ownership
type Ownership[P Package] struct {
	root            *Path
	packageDb       PackageDb[P]
	installPackages map[Package]bool
	brokenSymlink   map[*Path]bool
	orphanPathMap   map[*Path]bool
	mutex           sync.Mutex
}

func NewOwnership[P Package](
	root *Path,
	packageDb PackageDb[P],
) *Ownership[P] {
	return &Ownership[P]{
		root:            root,
		packageDb:       packageDb,
		installPackages: map[Package]bool{},
		brokenSymlink:   map[*Path]bool{},
		orphanPathMap:   map[*Path]bool{},
	}
}

func (o *Ownership[P]) skipPath(ctx context.Context, host types.Host, path *Path) (bool, bool, error) {
	logger := log.MustLogger(ctx)

	if path.IsDirectory() {
		logger.Debug("skipping", "path", path, "reason", "is directory")
		return true, false, nil
	}

	brokenSymlink := false
	if path.IsSymbolicLink() {
		targetPath, err := host.Readlink(ctx, path.String())
		if err != nil {
			return false, false, err
		}
		lstatPath := targetPath
		if !filepath.IsAbs(targetPath) {
			lstatPath = filepath.Dir(path.String()) + "/" + targetPath
		}

		stat_t, err := host.Lstat(ctx, lstatPath)
		if err != nil {
			if errors.Is(err, syscall.ENOENT) {
				brokenSymlink = true
			} else {
				return false, false, err
			}
		} else {
			if stat_t.Mode&syscall.S_IFMT == syscall.S_IFDIR {
				logger.Debug("skipping", "path", path, "reason", "symlink points to directory")
				return true, brokenSymlink, nil
			}
		}
	}

	return false, brokenSymlink, nil
}

func (o *Ownership[P]) processPath(ctx context.Context, host types.Host, path *Path) error {
	skip, brokenSymlink, err := o.skipPath(ctx, host, path)
	if err != nil {
		return err
	} else {
		if brokenSymlink {
			o.mutex.Lock()
			o.brokenSymlink[path] = true
			o.mutex.Unlock()
		}
		if skip {
			return nil
		}
	}

	name := path.String()
	if packages := o.packageDb.FindOwnerPackages(name); packages != nil {
		for _, pkg := range packages {
			o.mutex.Lock()
			o.installPackages[pkg] = true
			o.mutex.Unlock()
		}
	} else {
		matchingPackages := map[Package]bool{}
		var matchingPackage Package
		parentPath := name
		for {
			parentPath = filepath.Dir(parentPath)
			if parentPath == "/" {
				break
			}
			if packages := o.packageDb.FindOwnerPackages(parentPath); packages != nil {
				for _, pkg := range packages {
					matchingPackage = pkg
					matchingPackages[matchingPackage] = true
					if brokenSymlink {
						matchingPackage.AddBrokenSymLink(path.String())
					}
				}
			}
		}

		if len(matchingPackages) == 1 {
			matchingPackage.AddInferredOwnedPath(path.String())
		} else {
			o.mutex.Lock()
			o.orphanPathMap[path] = true
			o.mutex.Unlock()
		}
	}

	return nil
}

func (o *Ownership[P]) Compile(
	ctx context.Context,
	host types.Host,
) error {
	logger := log.MustLogger(ctx)

	logger.Info("Inferring file package ownership")

	ctx, cancel := context.WithCancel(ctx)

	limitCh := make(chan any, 128)
	var wg sync.WaitGroup
	errCh := make(chan error)

	for path := range o.root.ListRecursively() {
		limitCh <- true
		wg.Add(1)
		go func() {
			defer func() {
				<-limitCh
				wg.Done()
			}()
			if err := o.processPath(ctx, host, path); err != nil {
				errCh <- err
			}
		}()
	}

	doneCh := make(chan any)

	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case err := <-errCh:
		cancel()
		wg.Wait()
		return err
	case <-doneCh:
		cancel()
	}

	return nil
}

//gocyclo:ignore
func (o *Ownership[P]) CompileResources(
	ctx context.Context, host types.Host, resourcesPath string,
) error {
	logger := log.MustLogger(ctx)

	packages := []Package{}
	for pkg := range o.installPackages {
		packages = append(packages, pkg)
	}
	sort.SliceStable(packages, func(i, j int) bool {
		return packages[i].Name() < packages[j].Name()
	})

	resourcesByGroup := map[string]resourcesPkg.Resources{}
	for _, pkg := range packages {
		group, resources, issues, err := pkg.CompileResources(ctx, host)
		if err != nil {
			return err
		}
		if len(resources) > 0 {
			resourcesByGroup[group] = append(resourcesByGroup[group], resources)
		}
		if len(issues) > 0 {
			logger.Warn(
				"APT package has issues",
				"package", pkg.Name(),
				"issues", issues,
				"suggestion", "package may be corrupt, try reinstalling it; note that on debian/ubunt systems broken symlinks may be OK",
			)
		}
	}

	groups := []string{}
	for group := range resourcesByGroup {
		groups = append(groups, group)
	}
	sort.Strings(groups)

	packageResourcesPath := filepath.Join(resourcesPath, "packages")
	if err := os.MkdirAll(packageResourcesPath, 0700); err != nil {
		return err
	}
	for _, group := range groups {
		f, err := os.Create(filepath.Join(packageResourcesPath, fmt.Sprintf("%s.yaml", group)))
		if err != nil {
			return err
		}
		for _, resource := range resourcesByGroup[group] {
			encoder := yaml.NewEncoder(f)
			encoder.SetIndent(2)
			err := encoder.Encode(resource)
			err = errors.Join(err, encoder.Close())
			if err != nil {
				return errors.Join(err, f.Close())
			}
		}
		if err := f.Close(); err != nil {
			return err
		}
	}

	orphanPath := []*Path{}
	for path := range o.orphanPathMap {
		orphanPath = append(orphanPath, path)
	}
	sort.SliceStable(orphanPath, func(i int, j int) bool {
		return orphanPath[i].String() < orphanPath[j].String()
	})

	f, err := os.Create(filepath.Join(resourcesPath, "orphan_files.yaml"))
	if err != nil {
		return err
	}
	for _, path := range orphanPath {
		if o.brokenSymlink[path] {
			logger.Warn("ignoring orphan file that is broken symlink", "path", path.String())
			continue
		}
		file := &resourcesPkg.File{
			Path: path.String(),
		}
		if err := file.Load(ctx, host); err != nil {
			return errors.Join(err, f.Close())
		}
		encoder := yaml.NewEncoder(f)
		encoder.SetIndent(2)
		err := encoder.Encode(&resourcesPkg.Resources{file})
		err = errors.Join(err, encoder.Close())
		if err != nil {
			return errors.Join(err, f.Close())
		}
	}

	if err := f.Close(); err != nil {
		return err
	}

	return nil
}

type IndentWriter struct {
	w       io.Writer
	indent  string
	newLine bool
}

func NewIndentWriter(w io.Writer, indent string) *IndentWriter {
	return &IndentWriter{
		w:       w,
		indent:  indent,
		newLine: true,
	}
}

func (w *IndentWriter) Write(p []byte) (n int, err error) {
	var written int
	for _, b := range p {
		if w.newLine {
			if _, err := w.w.Write([]byte(w.indent)); err != nil {
				return written, err
			}
			w.newLine = false
		}

		if b == '\n' {
			w.newLine = true
		}

		if n, err := w.w.Write([]byte{b}); err != nil {
			return written + n, err
		} else {
			written += n
		}
	}
	return written, nil
}
