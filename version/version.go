package version

import (
	"runtime/debug"
)

// Version is the version of the built binary
type Version string

// IsCurrent returns whether Version is the same as the current binary.
func (v Version) IsCurrent() bool {
	return v == GetVersion()
}

// GetVersion returns the Version for the current binary.
func GetVersion() Version {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		panic("binaries not built with module support")
	}
	for _, buildSetting := range buildInfo.Settings {
		if buildSetting.Key == "vcs.revision" {
			return Version(buildSetting.Value)
		}
	}
	panic("vcs.revision not found at BuildInfo.Settings")
}
