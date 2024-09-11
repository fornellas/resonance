package discover

import (
	"context"
	"errors"
	"fmt"
	"io"
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

func (f *Ownership[P]) skipPath(ctx context.Context, host types.Host, path *Path) (bool, bool, error) {
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

func (f *Ownership[P]) processPath(ctx context.Context, host types.Host, path *Path) error {
	skip, brokenSymlink, err := f.skipPath(ctx, host, path)
	if err != nil {
		return err
	} else {
		if brokenSymlink {
			f.mutex.Lock()
			f.brokenSymlink[path] = true
			f.mutex.Unlock()
		}
		if skip {
			return nil
		}
	}

	name := path.String()
	if packages := f.packageDb.FindOwnerPackages(name); packages != nil {
		for _, pkg := range packages {
			f.mutex.Lock()
			f.installPackages[pkg] = true
			f.mutex.Unlock()
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
			if packages := f.packageDb.FindOwnerPackages(parentPath); packages != nil {
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
			f.mutex.Lock()
			f.orphanPathMap[path] = true
			f.mutex.Unlock()
		}
	}

	return nil
}

func (f *Ownership[P]) Compile(
	ctx context.Context,
	host types.Host,
) error {
	logger := log.MustLogger(ctx)

	logger.Info("Inferring file package ownership")

	ctx, cancel := context.WithCancel(ctx)

	limitCh := make(chan any, 128)
	var wg sync.WaitGroup
	errCh := make(chan error)

	for path := range f.root.ListRecursively() {
		limitCh <- true
		wg.Add(1)
		go func() {
			defer func() {
				<-limitCh
				wg.Done()
			}()
			if err := f.processPath(ctx, host, path); err != nil {
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
func (f *Ownership[P]) Report(ctx context.Context, host types.Host, writer io.Writer) error {
	fmt.Fprintf(writer, "Report\n")
	packages := []Package{}
	for pkg := range f.installPackages {
		packages = append(packages, pkg)
	}
	sort.SliceStable(packages, func(i, j int) bool {
		return packages[i].Name() < packages[j].Name()
	})

	resourcesByGroup := map[string]resourcesPkg.Resources{}
	for _, pkg := range packages {
		group, resources, issues, err := pkg.GetResources(ctx, host)
		if err != nil {
			return err
		}
		if len(resources) > 0 {
			resourcesByGroup[group] = append(resourcesByGroup[group], resources)
		}
		for _, issue := range issues {
			fmt.Fprintf(writer, "Warn: apt package: %s: %s\n", pkg.Name(), issue)
		}
	}

	groups := []string{}
	for group := range resourcesByGroup {
		groups = append(groups, group)
	}
	sort.Strings(groups)

	fmt.Fprintf(writer, "  apt packages\n")
	indentWriter := NewIndentWriter(writer, "      ")

	for _, group := range groups {
		fmt.Fprintf(writer, "    %s\n", group)
		for _, resource := range resourcesByGroup[group] {
			encoder := yaml.NewEncoder(indentWriter)
			encoder.SetIndent(2)
			err := encoder.Encode(resource)
			err = errors.Join(err, encoder.Close())
			if err != nil {
				return err
			}
		}
	}

	fmt.Fprintf(writer, "  orphan files\n")
	orphanPath := []*Path{}
	for path := range f.orphanPathMap {
		orphanPath = append(orphanPath, path)
	}
	sort.SliceStable(orphanPath, func(i int, j int) bool {
		return orphanPath[i].String() < orphanPath[j].String()
	})

	for _, path := range orphanPath {
		if f.brokenSymlink[path] {
			fmt.Fprintf(writer, "Warn: orphan file: broken symlink: %#v\n", path.String())
			continue
		}
		file := &resourcesPkg.File{
			Path: path.String(),
		}
		if err := file.Load(ctx, host); err != nil {
			return err
		}
		if file.RegularFile != nil {
			// FIXME
			file.RegularFile = new(string)
		}
		encoder := yaml.NewEncoder(indentWriter)
		encoder.SetIndent(2)
		err := encoder.Encode(&resourcesPkg.Resources{file})
		err = errors.Join(err, encoder.Close())
		if err != nil {
			return err
		}
	}

	// TODO orphanPath
	// if brokenSymlink
	// 	> warn
	// else
	// 	> include

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
