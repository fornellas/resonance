package draft

// HostState holds the state of all managed resources for a host.
type HostState struct {
	APTPackages APTPackages
	DpkgArch    *DpkgArch
	Files       Files
}
