package draft

// State holds the state of all managed resources for a host.
type State struct {
	APTPackages APTPackages
	DpkgArch    *DpkgArch
	Files       Files
}
