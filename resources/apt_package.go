package resources

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"fmt"
	"reflect"

	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
)

// APTPackage manages APT packages.
type APTPackage struct {
	// The name of the package
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#package
	Package string `yaml:"package"`
	// Architectures.
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#architecture
	Architectures []string `yaml:"architecture"`
	// Package version.
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	Version string `yaml:"version,omitempty"`
	// Whether to remove the package
	Absent bool `yaml:"absent,omitempty"`
}

var validDpkgPackageRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9+\-.]{1,}$`)

var validDpkgArchitectureRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]+$`)

func (a *APTPackage) Validate() error {
	// Package
	if !validDpkgPackageRegexp.MatchString(string(a.Package)) {
		return fmt.Errorf("invalid package: %#v", a.Package)
	}

	// Architectures
	for _, architecture := range a.Architectures {
		if !validDpkgArchitectureRegexp.MatchString(architecture) {
			return fmt.Errorf("invalid package: %#v", architecture)
		}
	}

	// Version
	if strings.HasSuffix(a.Version, "+") {
		return fmt.Errorf("`version` can't end in +: %s", a.Version)
	}
	if strings.HasSuffix(a.Version, "-") {
		return fmt.Errorf("`version` can't end in -: %s", a.Version)
	}

	return nil
}

func (a *APTPackage) Satisfies(resource Resource) bool {
	b, ok := resource.(*APTPackage)
	if !ok {
		panic("bug: not APTPackage")
	}

	if a.Version != "" && b.Version == "" {
		bCopy := *b
		b = &bCopy
		b.Version = a.Version
	}

	// FIXME Architectures

	return reflect.DeepEqual(a, b)
}

type APTPackages struct{}

var aptCachePackageRegexp = regexp.MustCompile(`^(.+):$`)
var aptCachePackageInstalledRegexp = regexp.MustCompile(`^  Installed: (.+)$`)
var aptCachePackageCandidateRegexp = regexp.MustCompile(`^  Candidate: (.+)$`)
var aptCacheUnableToLocateRegexp = regexp.MustCompile(`^N: Unable to locate package (.+)$`)

func (a *APTPackages) getAptPackages(resources Resources) []*APTPackage {
	aptPackages := make([]*APTPackage, len(resources))
	for i, resurce := range resources {
		aptPackage, ok := resurce.(*APTPackage)
		if !ok {
			panic("bug: Resource is not a APTPackage")
		}
		aptPackages[i] = aptPackage
	}
	return aptPackages
}

// FIXME Architectures
func (a *APTPackages) Load(ctx context.Context, hst types.Host, resources Resources) error {
	aptPackages := a.getAptPackages(resources)

	hostCmd := types.Cmd{
		Path: "apt-cache",
		Args: []string{"policy"},
	}
	for _, aptPackage := range aptPackages {
		hostCmd.Args = append(hostCmd.Args, string(aptPackage.Package))
	}

	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, hst, hostCmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		return fmt.Errorf(
			"failed to run '%s': %s\nstdout:\n%s\nstderr:\n%s",
			hostCmd.String(), waitStatus.String(), stdout, stderr,
		)
	}

	pkgInstalledMap := map[string]string{}
	pkgCandidateMap := map[string]string{}
	var pkg string
	// FIXME stream output with scanner
	for _, line := range strings.Split(stdout, "\n") {
		matches := aptCachePackageRegexp.FindStringSubmatch(line)
		if len(matches) == 2 {
			pkg = matches[1]
			continue
		}

		matches = aptCacheUnableToLocateRegexp.FindStringSubmatch(line)
		if len(matches) == 2 {
			return errors.New(line)
		}

		if pkg != "" {
			matches := aptCachePackageInstalledRegexp.FindStringSubmatch(line)
			if len(matches) == 2 {
				pkgInstalledMap[pkg] = matches[1]
				continue
			}
			matches = aptCachePackageCandidateRegexp.FindStringSubmatch(line)
			if len(matches) == 2 {
				pkgCandidateMap[pkg] = matches[1]
				continue
			}
		}
	}

	for _, aptPackage := range aptPackages {
		installedVersion, ok := pkgInstalledMap[string(aptPackage.Package)]
		if !ok {
			return fmt.Errorf(
				"failed to get %#v package version: %#v: no version found on output:\n%s",
				aptPackage.Package, hostCmd.String(), stdout,
			)
		}

		if installedVersion == "(none)" {
			aptPackage.Absent = true
		} else {
			aptPackage.Version = installedVersion
		}
	}

	return nil
}

func (a *APTPackages) Resolve(ctx context.Context, hst types.Host, resources Resources) error {
	return nil
}

// FIXME Architectures
func (a *APTPackages) Apply(ctx context.Context, hst types.Host, resources Resources) error {
	aptPackages := a.getAptPackages(resources)

	// Package arguments
	pkgArgs := make([]string, len(aptPackages))
	for i, aptPackage := range aptPackages {
		var pkgArg string
		if aptPackage.Absent {
			pkgArg = fmt.Sprintf("%s-", aptPackage.Package)
		} else {
			pkgArg = fmt.Sprintf("%s=%s", aptPackage.Package, aptPackage.Version)
		}
		pkgArgs[i] = pkgArg
	}

	// Run apt
	cmd := types.Cmd{
		Path: "apt-get",
		Args: append([]string{"--yes", "install"}, pkgArgs...),
	}
	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, hst, cmd)
	if err != nil {
		return fmt.Errorf("failed to run '%s': %s", cmd, err)
	}
	if !waitStatus.Success() {
		return fmt.Errorf(
			"failed to run '%v': %s:\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}

	return nil
}

func init() {
	RegisterGroupResource(
		reflect.TypeOf((*APTPackage)(nil)).Elem(),
		reflect.TypeOf((*APTPackages)(nil)).Elem(),
	)
}
