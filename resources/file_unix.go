package resources

import "path/filepath"

// Returns true if given path is a Unix path that's absolute.
func isAbsUnixPath(path string) bool {
	return filepath.IsAbs(path)
}

// Returns true if given path is a Unix path that's clean.
func isCleanUnixPath(path string) bool {
	return (filepath.Clean(path) == path)
}

// filepath.Dir for unix.
func dirUnix(path string) string {
	return filepath.Dir(path)
}
