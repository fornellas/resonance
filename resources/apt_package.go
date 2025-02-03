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

// AptPackageState manages APT packages.
type AptPackageState struct {
	// The name of the package
	Package string `yaml:"package"`
	// Whether to remove the package
	Absent bool `yaml:"absent,omitempty"`
	// Package version
	Version string `yaml:"version,omitempty"`
}

// https://www.debian.org/doc/debian-policy/ch-controlfields.html#package
var validAptPackageNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9+\-.]{1,}$`)

func (s *AptPackageState) Validate() error {
	// Package
	if !validAptPackageNameRegexp.MatchString(string(s.Package)) {
		return fmt.Errorf("`package` must match regexp %s: %s", validAptPackageNameRegexp, s.Version)
	}

	// Version
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	if strings.HasSuffix(s.Version, "+") {
		return fmt.Errorf("`version` can't end in +: %s", s.Version)
	}
	if strings.HasSuffix(s.Version, "-") {
		return fmt.Errorf("`version` can't end in -: %s", s.Version)
	}

	return nil
}

func (s *AptPackageState) Satisfies(b *AptPackageState) bool {

	if s.Version != "" && b.Version == "" {
		bCopy := *b
		b = &bCopy
		b.Version = s.Version
	}

	return reflect.DeepEqual(s, b)
}

type AptPackageProvisioner struct {
	Host types.Host
}

func NewAptPackageProvisioner(host types.Host) (*AptPackageProvisioner, error) {
	return &AptPackageProvisioner{
		Host: host,
	}, nil
}

var aptCachePackageRegexp = regexp.MustCompile(`^(.+):$`)
var aptCachePackageInstalledRegexp = regexp.MustCompile(`^  Installed: (.+)$`)
var aptCachePackageCandidateRegexp = regexp.MustCompile(`^  Candidate: (.+)$`)
var aptCacheUnableToLocateRegexp = regexp.MustCompile(`^N: Unable to locate package (.+)$`)

func (p *AptPackageProvisioner) Load(ctx context.Context, targetStates []*AptPackageState) error {
	hostCmd := types.Cmd{
		Path: "apt-cache",
		Args: []string{"policy"},
	}
	for _, targetState := range targetStates {
		hostCmd.Args = append(hostCmd.Args, string(targetState.Package))
	}

	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, p.Host, hostCmd)
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

	for _, aptPackage := range targetStates {
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

func (p *AptPackageProvisioner) Apply(ctx context.Context, targetStates []*AptPackageState) error {
	// Package arguments
	pkgArgs := make([]string, len(targetStates))
	for i, aptPackage := range targetStates {
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
	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, p.Host, cmd)
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
