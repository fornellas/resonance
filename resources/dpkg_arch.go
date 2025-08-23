package resources

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
)

// DpkgArch manages the set of foreign architectures that dpkg is configured to support.
// This allows installing packages built for architectures other than the system's native one,
// enabling multiarch support as described in https://wiki.debian.org/Multiarch/HOWTO.
//
// The ForeignArchitectures field lists all extra architectures to be enabled.
// When architectures are removed, all packages for that architecture are purged before removal.
type DpkgArch struct {
	// ForeignArchitectures specifies extra architectures dpkg is configured to allow packages to be installed for.
	ForeignArchitectures []string
}

// Satisfies returns true only when a satisfies b.
func (a *DpkgArch) Satisfies(b *DpkgArch) bool {
	for _, bArch := range b.ForeignArchitectures {
		if !slices.Contains(a.ForeignArchitectures, bArch) {
			return false
		}
	}
	return true
}

func (a *DpkgArch) Validate() error {
	for _, architecture := range a.ForeignArchitectures {
		if !validDpkgArchitectureRegexp.MatchString(architecture) {
			return fmt.Errorf("invalid architecture: %#v", architecture)
		}
	}
	return nil
}
func getSystemArch(ctx context.Context, hst types.Host) (string, error) {
	cmd := types.Cmd{
		Path: "/usr/bin/dpkg",
		Args: []string{"--print-architecture"},
	}
	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, hst, cmd)
	if err != nil {
		return "", err
	}
	if !waitStatus.Success() {
		return "", fmt.Errorf("failed to get system architecture: %s\nstdout:\n%s\nstderr:\n%s", waitStatus.String(), stdout, stderr)
	}
	return strings.TrimSpace(stdout), nil
}

func getCurrentForeignArchs(ctx context.Context, hst types.Host) (map[string]struct{}, error) {
	cmd := types.Cmd{
		Path: "/usr/bin/dpkg",
		Args: []string{"--print-foreign-architectures"},
	}
	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, hst, cmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		return nil, fmt.Errorf("failed to get current foreign architectures: %s\nstdout:\n%s\nstderr:\n%s", waitStatus.String(), stdout, stderr)
	}
	archs := map[string]struct{}{}
	for _, arch := range slices.DeleteFunc(strings.Split(strings.TrimSpace(stdout), "\n"), func(s string) bool { return s == "" }) {
		archs[arch] = struct{}{}
	}
	return archs, nil
}

func getDesiredArchs(foreignArchs []string, systemArch string) (map[string]struct{}, error) {
	desired := map[string]struct{}{}
	for _, arch := range foreignArchs {
		if arch == systemArch {
			return nil, fmt.Errorf("foreign architecture %q matches system architecture", arch)
		}
		desired[arch] = struct{}{}
	}
	return desired, nil
}

func addMissingArchs(ctx context.Context, hst types.Host, current, desired map[string]struct{}) error {
	for arch := range desired {
		if _, ok := current[arch]; !ok {
			cmd := types.Cmd{
				Path: "/usr/bin/dpkg",
				Args: []string{"--add-architecture", arch},
			}
			waitStatus, _, stderr, err := lib.SimpleRun(ctx, hst, cmd)
			if err != nil {
				return err
			}
			if !waitStatus.Success() {
				return fmt.Errorf("failed to add architecture %s: %s\nstderr:\n%s", arch, waitStatus.String(), stderr)
			}
		}
	}
	return nil
}

func removeExtraArchs(ctx context.Context, hst types.Host, current, desired map[string]struct{}) error {
	for arch := range current {
		if _, ok := desired[arch]; !ok {
			// Purge all packages for this architecture before removing it
			purgeCmd := types.Cmd{
				Path: "/usr/bin/apt",
				Args: []string{"-y", "--allow-remove-essential", "purge", "*:" + arch},
			}
			waitStatus, _, stderr, err := lib.SimpleRun(ctx, hst, purgeCmd)
			if err != nil {
				return err
			}
			// Ignore error if no packages for the architecture exist
			if !waitStatus.Success() && !strings.Contains(stderr, "Unable to locate package") {
				return fmt.Errorf("failed to purge packages for architecture %s: %s\nstderr:\n%s", arch, waitStatus.String(), stderr)
			}

			cmd := types.Cmd{
				Path: "/usr/bin/dpkg",
				Args: []string{"--remove-architecture", arch},
			}
			waitStatus, _, stderr, err = lib.SimpleRun(ctx, hst, cmd)
			if err != nil {
				return err
			}
			if !waitStatus.Success() {
				return fmt.Errorf("failed to remove architecture %s: %s\nstderr:\n%s", arch, waitStatus.String(), stderr)
			}
		}
	}
	return nil
}

func (a *DpkgArch) Apply(ctx context.Context, hst types.Host) error {
	systemArch, err := getSystemArch(ctx, hst)
	if err != nil {
		return err
	}

	foreignArchsMap, err := getCurrentForeignArchs(ctx, hst)
	if err != nil {
		return err
	}

	desiredArchs, err := getDesiredArchs(a.ForeignArchitectures, systemArch)
	if err != nil {
		return err
	}

	if err := addMissingArchs(ctx, hst, foreignArchsMap, desiredArchs); err != nil {
		return err
	}

	if err := removeExtraArchs(ctx, hst, foreignArchsMap, desiredArchs); err != nil {
		return err
	}

	return nil
}
