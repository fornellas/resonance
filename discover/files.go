package discover

import (
	"context"
	"sort"
	"sync"
	"syscall"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

type Path struct {
	types.DirEnt
	parent   *Path
	children []*Path
}

func (f *Path) FileCount() int {
	len := 1
	for _, child := range f.children {
		len += child.FileCount()
	}
	return len
}

func (f *Path) String() string {
	path := f.DirEnt.Name
	parent := f.parent
	for {
		if parent == nil {
			break
		}
		if parent.DirEnt.Name == "/" {
			path = parent.DirEnt.Name + path
		} else {
			path = parent.DirEnt.Name + "/" + path
		}
		parent = parent.parent
	}
	return path
}

func (f *Path) ListRecursively() chan *Path {
	ch := make(chan *Path)
	go func() {
		defer close(ch)
		ch <- f
		for _, child := range f.children {
			for path := range child.ListRecursively() {
				ch <- path
			}
		}
	}()
	return ch
}

func (f *Path) load(
	ctx context.Context,
	host types.Host,
	ignorePatterns []string,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if f.DirEnt.Name != "/" && (!f.DirEnt.IsDirectory() || f.DirEnt.IsSymbolicLink()) {
		return nil
	}

	logger := log.MustLogger(ctx)
	name := f.String()

	ctx, cancel := context.WithCancel(ctx)

	dirEntResultCh, cancelReadDir := host.ReadDir(ctx, name)
	defer cancelReadDir()

	limitCh := make(chan any, 128)
	var wg sync.WaitGroup
	errCh := make(chan error)
	var mutex sync.Mutex

	for dirEntResult := range dirEntResultCh {
		limitCh <- true
		wg.Add(1)

		go func() {
			defer func() {
				<-limitCh
				wg.Done()
			}()
			if dirEntResult.Error != nil {
				errCh <- dirEntResult.Error
				return
			}

			childPatht := &Path{
				DirEnt: dirEntResult.DirEnt,
				parent: f,
			}
			childPath := childPatht.String()
			for _, pattern := range ignorePatterns {
				ignore, err := doublestar.PathMatch(pattern, childPath)
				if err != nil {
					errCh <- err
					return
				}
				if ignore {
					logger.Debug("Ignoring", "path", childPath)
					return
				}
			}

			mutex.Lock()
			f.children = append(f.children, childPatht)
			mutex.Unlock()

			if err := childPatht.load(ctx, host, ignorePatterns); err != nil {
				errCh <- err
				return
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

	sort.SliceStable(f.children, func(i, j int) bool {
		return f.children[i].Name < f.children[j].Name
	})

	return nil
}

func LoadRoot(
	ctx context.Context,
	host types.Host,
	ignorePatterns []string,
) (*Path, error) {
	ctx, _ = log.MustContextLoggerWithSection(ctx, "Finding all files")

	path := Path{
		DirEnt: types.DirEnt{
			Type: syscall.DT_DIR,
			Name: "/",
		},
	}

	if err := path.load(ctx, host, ignorePatterns); err != nil {
		return nil, err
	}

	return &path, nil
}
