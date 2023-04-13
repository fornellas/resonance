package resource

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/resource"
)

// APTPackageState is State for APTPackage
type APTPackageState struct {
	// Package version
	Version string `yaml:"version"`
}

func (aps APTPackageState) ValidateAndUpdate(ctx context.Context, hst host.Host) (resource.State, error) {
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

func (ap APTPackage) ValidateName(name resource.Name) error {
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#source
	return nil
}

func (ap APTPackage) Diff(a, b resource.State) resource.Chunks {
	if a != nil && b != nil {
		aptPackageStateA := a.(APTPackageState)
		aptPackageStateB := b.(APTPackageState)
		if aptPackageStateB.Version == "" {
			aptPackageStateB.Version = aptPackageStateA.Version
		}
		return resource.DiffAsYaml(aptPackageStateA, aptPackageStateB)
	} else {
		return resource.DiffAsYaml(a, b)
	}
}

var aptCachePackageRegexp = regexp.MustCompile(`^(.+):$`)
var aptCachePackageInstalledRegexp = regexp.MustCompile(`^  Installed: (.+)$`)
var aptCachePackageCandidateRegexp = regexp.MustCompile(`^  Candidate: (.+)$`)
var aptCacheUnableToLocateRegexp = regexp.MustCompile(`^N: Unable to locate package (.+)$`)

func (ap APTPackage) GetStates(
	ctx context.Context, hst host.Host, names resource.Names,
) (map[resource.Name]resource.State, error) {
	hostCmd := types.Cmd{
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

	nameStateMap := map[resource.Name]resource.State{}
	for _, name := range names {
		installedVersion, ok := pkgInstalledMap[string(name)]
		if !ok {
			return nil, fmt.Errorf(
				"failed to get %s package version: %s:\n%s",
				name, hostCmd.String(), stdout,
			)
		}

		var state resource.State
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
	ctx context.Context, hst host.Host, actionNameStateMap map[resource.Action]map[resource.Name]resource.State,
) error {
	// Package arguments
	pkgs := []string{}
	for action, nameStateMap := range actionNameStateMap {
		var pkgAction string
		switch action {
		case resource.ActionOk:
		case resource.ActionConfigure:
			pkgAction = ""
		case resource.ActionDestroy:
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
	cmd := types.Cmd{
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
	resource.MergeableManageableResourcesTypeMap["APTPackage"] = APTPackage{}
	resource.ManageableResourcesStateMap["APTPackage"] = APTPackageState{}
}
