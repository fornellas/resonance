package draft

// State holds the state of all managed resources for a host.
type State struct {
	APTPackage []APTPackage
	DpkgArch   *DpkgArch
	File       []File
}
