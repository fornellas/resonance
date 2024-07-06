package resources

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"fmt"
	"reflect"

	"github.com/fornellas/resonance/host"
)

// APTPackage manages APT packages.
type APTPackage struct {
	// The name of the package
	Package string `yaml:"package"`
	// Whether to remove the package
	Remove bool `yaml:"remove"`
	// Package version
	Version string `yaml:"version"`
}

// https://www.debian.org/doc/debian-policy/ch-controlfields.html#package
var validAptPackageNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9+\-.]{1,}$`)

func (a *APTPackage) Validate() error {
	// Package
	if !validAptPackageNameRegexp.MatchString(a.Package) {
		return fmt.Errorf("`package` must match regexp %s: %s", validAptPackageNameRegexp, a.Version)
	}

	// Remove
	if a.Remove {
		if a.Version != "" {
			return fmt.Errorf("'version' can not be set when 'remove' is true")
		}
	}

	// Version
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	if strings.HasSuffix(a.Version, "+") {
		return fmt.Errorf("`version` can't end in +: %s", a.Version)
	}
	if strings.HasSuffix(a.Version, "-") {
		return fmt.Errorf("`version` can't end in -: %s", a.Version)
	}

	return nil
}

func (a *APTPackage) Name() string {
	return a.Package
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

func (a *APTPackages) Load(ctx context.Context, hst host.Host, resources Resources) error {
	aptPackages := a.getAptPackages(resources)

	hostCmd := host.Cmd{
		Path: "apt-cache",
		Args: []string{"policy"},
	}
	for _, aptPackage := range aptPackages {
		hostCmd.Args = append(hostCmd.Args, aptPackage.Package)
	}

	waitStatus, stdout, stderr, err := host.Run(ctx, hst, hostCmd)
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
		installedVersion, ok := pkgInstalledMap[aptPackage.Package]
		if !ok {
			return fmt.Errorf(
				"failed to get %s package version: %s:\n%s",
				aptPackage.Package, hostCmd.String(), stdout,
			)
		}

		if installedVersion == "(none)" {
			aptPackage.Remove = true
		} else {
			aptPackage.Version = installedVersion
		}
	}

	return nil
}

func (a *APTPackages) Update(ctx context.Context, hst host.Host, resources Resources) error {
	return nil
}

func (a *APTPackages) Apply(ctx context.Context, hst host.Host, resources Resources) error {
	aptPackages := a.getAptPackages(resources)

	// Package arguments
	pkgArgs := make([]string, len(aptPackages))
	for i, aptPackage := range aptPackages {
		var pkgArg string
		if aptPackage.Remove {
			pkgArg = fmt.Sprintf("%s-", aptPackage.Package)
		} else {
			pkgArg = fmt.Sprintf("%s=%s", aptPackage.Package, aptPackage.Version)
		}
		pkgArgs[i] = pkgArg
	}

	// Run apt
	cmd := host.Cmd{
		Path: "apt-get",
		Args: append([]string{"--yes", "install"}, pkgArgs...),
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, hst, cmd)
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
