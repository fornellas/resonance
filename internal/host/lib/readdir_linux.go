package lib

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/fornellas/resonance/host"
)

// Implements Host.Run for Linux locahost.
func ReadDir(ctx context.Context, name string) (<-chan host.DirEntResult, func()) {
	ctx, cancel := context.WithCancel(ctx)

	dirEntResultCh := make(chan host.DirEntResult, 100)

	go func() {
		if !filepath.IsAbs(name) {
			dirEntResultCh <- host.DirEntResult{
				Error: &fs.PathError{
					Op:   "ReadDir",
					Path: name,
					Err:  errors.New("path must be absolute"),
				},
			}
			close(dirEntResultCh)
			return
		}

		fd, err := syscall.Open(name, syscall.O_RDONLY, 0)
		if err != nil {
			dirEntResultCh <- host.DirEntResult{Error: err}
			close(dirEntResultCh)
			return
		}
		defer syscall.Close(fd)

		buf := make([]byte, 8196)

		for {
			// We do this via syscall.Getdents instead of os.ReadDir, because the latter
			// requires doing aditional stat calls, which is slower.
			n, err := syscall.Getdents(fd, buf)
			if err != nil {
				dirEntResultCh <- host.DirEntResult{Error: err}
				break
			}

			if n == 0 {
				break
			}

			offset := 0
			for offset < n {
				dirent := (*syscall.Dirent)(unsafe.Pointer(&buf[offset]))

				var l int
				for l = 0; l < len(dirent.Name); l++ {
					if dirent.Name[l] == 0 {
						break
					}
				}
				nameBytes := make([]byte, l)
				for i := 0; i < l; i++ {
					nameBytes[i] = byte(dirent.Name[i])
				}
				name := string(nameBytes)

				if name != "." && name != ".." {
					dirEnt := host.DirEnt{
						Ino:  dirent.Ino,
						Type: dirent.Type,
						Name: name,
					}

					select {
					case dirEntResultCh <- host.DirEntResult{DirEnt: dirEnt}:
					case <-ctx.Done():
						close(dirEntResultCh)
						return
					}
				}

				offset += int(dirent.Reclen)
			}
		}

		close(dirEntResultCh)
	}()

	return dirEntResultCh, cancel
}
