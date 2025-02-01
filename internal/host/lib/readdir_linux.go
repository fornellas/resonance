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
func ReadDir(ctx context.Context, name string) ([]host.DirEnt, error) {
	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "ReadDir",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	fd, err := syscall.Open(name, syscall.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(fd)

	buf := make([]byte, 4096)

	dirEnts := []host.DirEnt{}
	for {
		// We do this via syscall.Getdents instead of os.ReadDir, because the latter
		// requires doing aditional stat calls, which is slower.
		n, err := syscall.Getdents(fd, buf)
		if err != nil {
			return nil, err
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
				dirEnts = append(dirEnts, host.DirEnt{
					Ino:  dirent.Ino,
					Type: dirent.Type,
					Name: name,
				})
			}

			offset += int(dirent.Reclen)
		}
	}

	return dirEnts, nil
}
