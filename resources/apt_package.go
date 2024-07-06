package resources

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/fornellas/resonance/diff"
	"github.com/fornellas/resonance/host"
)

// APTPackageState is State for APTPackage
type APTPackageState struct {
	// Package version
	Version string `yaml:"version"`
}

func (aps APTPackageState) ValidateAndUpdate(ctx context.Context, hst host.Host) (State, error) {
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	if strings.HasSuffix(aps.Version, "+") {
		return nil, fmt.Errorf("version can't end in +: %s", aps.Version)
	}
	if strings.HasSuffix(aps.Version, "-") {
		return nil, fmt.Errorf("version can't end in -: %s", aps.Version)
	}
	return aps, nil
}

// APTPackage resource manages files.
type APTPackage struct{}

func (ap APTPackage) ValidateName(name Name) error {
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#source
	return nil
}

func (ap APTPackage) Diff(a, b State) diff.Chunks {
	if a != nil && b != nil {
		aptPackageStateA := a.(APTPackageState)
		aptPackageStateB := b.(APTPackageState)
		if aptPackageStateB.Version == "" {
			aptPackageStateB.Version = aptPackageStateA.Version
		}
		return diff.DiffAsYaml(aptPackageStateA, aptPackageStateB)
	} else {
		return diff.DiffAsYaml(a, b)
	}
}

var aptCachePackageRegexp = regexp.MustCompile(`^(.+):$`)
var aptCachePackageInstalledRegexp = regexp.MustCompile(`^  Installed: (.+)$`)
var aptCachePackageCandidateRegexp = regexp.MustCompile(`^  Candidate: (.+)$`)
var aptCacheUnableToLocateRegexp = regexp.MustCompile(`^N: Unable to locate package (.+)$`)

func (ap APTPackage) GetStates(
	ctx context.Context, hst host.Host, names Names,
) (map[Name]State, error) {
	hostCmd := host.Cmd{
		Path: "apt-cache",
		Args: []string{"policy"},
	}
	for _, name := range names {
		hostCmd.Args = append(hostCmd.Args, string(name))
	}

	waitStatus, stdout, stderr, err := host.Run(ctx, hst, hostCmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		return nil, fmt.Errorf(
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
			return nil, errors.New(line)
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

	nameStateMap := map[Name]State{}
	for _, name := range names {
		installedVersion, ok := pkgInstalledMap[string(name)]
		if !ok {
			return nil, fmt.Errorf(
				"failed to get %s package version: %s:\n%s",
				name, hostCmd.String(), stdout,
			)
		}

		var state State
		if installedVersion != "(none)" {
			state = APTPackageState{
				Version: installedVersion,
			}
		}
		nameStateMap[name] = state
	}
	return nameStateMap, nil
}

func (ap APTPackage) ConfigureAll(
	ctx context.Context, hst host.Host, actionNameStateMap map[Action]map[Name]State,
) error {
	// Package arguments
	pkgs := []string{}
	for action, nameStateMap := range actionNameStateMap {
		var pkgAction string
		switch action {
		case ActionOk:
		case ActionConfigure:
			pkgAction = ""
		case ActionDestroy:
			pkgAction = "-"
		default:
			return fmt.Errorf("unexpected action %s", action)
		}
		for name, state := range nameStateMap {
			var version string
			if state != nil {
				aptPackageState := state.(APTPackageState)
				if aptPackageState.Version != "" {
					version = fmt.Sprintf("=%s", aptPackageState.Version)
				}
			}
			pkgs = append(pkgs, fmt.Sprintf("%s%s%s", string(name), version, pkgAction))
		}
	}

	// Run apt
	cmd := host.Cmd{
		Path: "apt-get",
		Args: append([]string{"--yes", "install"}, pkgs...),
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
	MergeableResourcesTypeMap["APTPackage"] = APTPackage{}
	ResourcesStateMap["APTPackage"] = APTPackageState{}
}
