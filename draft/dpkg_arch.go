package draft

// DpkgArch manages the set of foreign architectures that dpkg is configured to support.
// This allows installing packages built for architectures other than the system's native one,
// enabling multiarch support as described in https://wiki.debian.org/Multiarch/HOWTO.
//
// The ForeignArchitectures field lists all extra architectures to be enabled.
type DpkgArch struct {
	// ForeignArchitectures specifies extra architectures dpkg is configured to allow packages to be
	//  installed for. Required.
	ForeignArchitectures []string
}
