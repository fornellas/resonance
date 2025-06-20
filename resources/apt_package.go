package resources

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"fmt"
	"reflect"

	"github.com/fornellas/resonance/concurrency"
	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
)

// A debconf question.
// See https://wiki.debian.org/debconf
type DebconfQuestion string

// Debconf selections for a DebconfQuestion.
// See https://wiki.debian.org/debconf
type DebconfSelection struct {
	Answer string `yaml:"answer"`
	Seen   bool   `yaml:"seen,omitempty"`
}

// APTPackage manages APT packages.
type APTPackage struct {
	// The name of the package
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#package
	Package string `yaml:"package"`
	// Whether to remove the package
	Absent bool `yaml:"absent,omitempty"`
	// Architectures.
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#architecture
	Architectures []string `yaml:"architectures,omitempty"`
	// Package version.
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	Version string `yaml:"version,omitempty"`
	// Package debconf selections.
	// See https://wiki.debian.org/debconf
	DebconfSelections map[DebconfQuestion]DebconfSelection `yaml:"debconf,omitempty"`
}

var validDpkgPackageRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9+\-.]{1,}$`)

var validDpkgArchitectureRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]+$`)

var validDpkgVersionRegexp = regexp.MustCompile(`^(?:([0-9]+):)?(([0-9][A-Za-z0-9.+~]*)|([0-9][A-Za-z0-9.+~-]*-[A-Za-z0-9+.~]+))$`)

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
	if a.Version != "" && !validDpkgVersionRegexp.MatchString(a.Version) {
		return fmt.Errorf("invalid version: %#v", a.Version)
	}

	return nil
}

func (a *APTPackage) Satisfies(resource Resource) bool {
	b, ok := resource.(*APTPackage)
	if !ok {
		panic("bug: not APTPackage")
	}

	if a.Package != b.Package {
		return false
	}

	if a.Absent != b.Absent {
		return false
	}

	for _, arch := range a.Architectures {
		if !slices.Contains(b.Architectures, arch) {
			return false
		}
	}

	if len(a.Version) > 0 {
		if len(b.Version) > 0 {
			if a.Version != b.Version {
				return false
			}
		}
	} else {
		if len(b.Version) > 0 {
			return false
		}
	}

	for debconfQuestion, debconfSelection := range b.DebconfSelections {
		if bDebconfSelection, ok := a.DebconfSelections[debconfQuestion]; ok {
			if debconfSelection.Answer != bDebconfSelection.Answer {
				return false
			}
			if debconfSelection.Seen != bDebconfSelection.Seen {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

type APTPackages struct{}

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

var debconfShowRegexp = regexp.MustCompile("^([ *]) (.+):(| (.+))$")

func (a *APTPackages) preparePackageQueries(
	aptPackages []*APTPackage,
) ([]string, map[string]*APTPackage) {
	packageQueries := make([]string, 0)
	packageToResource := make(map[string]*APTPackage)

	for _, aptPackage := range aptPackages {
		if len(aptPackage.Architectures) > 0 {
			for _, arch := range aptPackage.Architectures {
				query := fmt.Sprintf("%s:%s", aptPackage.Package, arch)
				packageQueries = append(packageQueries, query)
			}
		} else {
			packageQueries = append(packageQueries, aptPackage.Package)
		}
		packageToResource[aptPackage.Package] = aptPackage
		aptPackage.Absent = true
		aptPackage.Version = ""
		aptPackage.Architectures = nil
		aptPackage.DebconfSelections = nil
	}

	return packageQueries, packageToResource
}

func (a *APTPackages) runDpkgQuery(ctx context.Context, hst types.Host, packageQueries []string, resourceCount int) (string, error) {
	args := []string{
		"--show",
		"--showformat=Package=${Package}\nArchitecture=${Architecture}\nVersion=${Version}\nend\n",
	}
	args = append(args, packageQueries...)

	cmd := types.Cmd{
		Path: "/usr/bin/dpkg-query",
		Args: args,
	}

	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, hst, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to run dpkg-query: %w\n%s\nSTDOUT:\n%s\nSTDERR:\n%s", err, cmd, stdout, stderr)
	}

	if waitStatus.Exited && waitStatus.ExitCode == 1 && resourceCount == 1 && strings.HasPrefix(stdout, "dpkg-query: no packages found matching ") {
		return "", nil
	}

	return stdout, nil
}

func (a *APTPackages) processDpkgOutput(stdout string, packageToResource map[string]*APTPackage) error {
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	currentPkg := ""
	currentArch := ""
	currentVersion := ""

	for scanner.Scan() {
		line := scanner.Text()
		if line == "end" {
			if aptPackage, ok := packageToResource[currentPkg]; ok {
				aptPackage.Absent = false
				aptPackage.Version = currentVersion
				if currentArch != "" && !slices.Contains(aptPackage.Architectures, currentArch) {
					aptPackage.Architectures = append(aptPackage.Architectures, currentArch)
				}
			}
			currentPkg = ""
			currentArch = ""
			currentVersion = ""
		} else if prefix, found := strings.CutPrefix(line, "Package="); found {
			currentPkg = prefix
		} else if prefix, found := strings.CutPrefix(line, "Architecture="); found {
			currentArch = prefix
		} else if prefix, found := strings.CutPrefix(line, "Version="); found {
			currentVersion = prefix
		} else {
			return fmt.Errorf("failed to process dpkg-query stdout line: %s", line)
		}
	}

	return scanner.Err()
}
func (a *APTPackages) debconfCommunicate(
	ctx context.Context,
	hst types.Host,
	pkg, command string,
) (string, error) {
	stdinReader := strings.NewReader(command)
	stdoutBuffer := bytes.Buffer{}
	stderrBuffer := bytes.Buffer{}
	cmd := types.Cmd{
		Path:   "debconf-communicate",
		Args:   []string{pkg},
		Env:    []string{"DEBIAN_FRONTEND=noninteractive"},
		Stdin:  stdinReader,
		Stdout: &stdoutBuffer,
		Stderr: &stderrBuffer,
	}
	waitStatus, err := hst.Run(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("%s failed: %w", cmd, err)
	}
	stdout := stdoutBuffer.String()
	stderr := stderrBuffer.String()
	if !waitStatus.Success() {
		return "", fmt.Errorf("%s failed: %s\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), stdout, stderr)
	}
	if len(stderr) > 0 {
		return "", fmt.Errorf("%s failed: %s\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), stdout, stderr)
	}
	var value string
	if strings.HasPrefix(stdout, "0 ") {
		value = strings.TrimSuffix(stdout[2:], "\n")
	} else {
		if stdout != "0" {
			return "", fmt.Errorf("%s failed: %s\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), stdout, stderr)
		}
	}

	return value, nil
}

func (a *APTPackages) loadDebconfSelections(ctx context.Context, hst types.Host, aptPackages []*APTPackage) error {
	concurrencyGroup := concurrency.NewConcurrencyGroup(ctx)

	for _, aptPackage := range aptPackages {
		if aptPackage.Absent {
			continue
		}

		concurrencyGroup.Run(func() error {
			cmd := types.Cmd{
				Path: "debconf-show",
				Args: []string{aptPackage.Package},
			}
			waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, hst, cmd)
			if err != nil {
				return err
			}
			if !waitStatus.Success() {
				return fmt.Errorf("%s failed: %s\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), stdout, stderr)
			}

			if len(stderr) > 0 {
				return fmt.Errorf("%s failed:\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, stdout, stderr)
			}

			scanner := bufio.NewScanner(strings.NewReader(stdout))
			for scanner.Scan() {
				line := scanner.Text()
				matches := debconfShowRegexp.FindStringSubmatch(line)
				if matches == nil {
					return fmt.Errorf("%s failed: can not parse debconf-show output line: %s", cmd, line)
				}

				seen := matches[1] == "*"
				question := DebconfQuestion(matches[2])
				answer := matches[4]
				if answer == "(password omitted)" {
					command := fmt.Sprintf("get %s\n", question)
					answer, err = a.debconfCommunicate(ctx, hst, aptPackage.Package, command)
					if err != nil {
						return err
					}
				}
				if aptPackage.DebconfSelections == nil {
					aptPackage.DebconfSelections = map[DebconfQuestion]DebconfSelection{}
				}
				aptPackage.DebconfSelections[question] = DebconfSelection{
					Answer: answer,
					Seen:   seen,
				}
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("%s failed: can not scan stderr: %w\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, err, stdout, stderr)
			}
			return nil
		})
	}
	if err := errors.Join(concurrencyGroup.Wait()...); err != nil {
		return err
	}
	return nil
}

func (a *APTPackages) Load(ctx context.Context, hst types.Host, resources Resources) error {
	aptPackages := a.getAptPackages(resources)

	packageQueries, packageToResource := a.preparePackageQueries(aptPackages)

	stdout, err := a.runDpkgQuery(ctx, hst, packageQueries, len(resources))
	if err != nil {
		return err
	}

	if err := a.processDpkgOutput(stdout, packageToResource); err != nil {
		return fmt.Errorf("failed scanning dpkg-query output: %w", err)
	}

	if err := a.loadDebconfSelections(ctx, hst, aptPackages); err != nil {
		return fmt.Errorf("failed loading debconf: %w", err)
	}

	return nil
}

func (a *APTPackages) Resolve(ctx context.Context, hst types.Host, resources Resources) error {
	return nil
}

func (a *APTPackages) Apply(ctx context.Context, hst types.Host, resources Resources) error {
	aptPackages := a.getAptPackages(resources)

	pkgArgs := []string{}
	for _, aptPackage := range aptPackages {
		if aptPackage.Absent {
			pkgArgs = append(pkgArgs, fmt.Sprintf("%s-", aptPackage.Package))
		} else {
			var pkgArg string
			if len(aptPackage.Version) > 0 {
				pkgArg = fmt.Sprintf("%s=%s", aptPackage.Package, aptPackage.Version)
			} else {
				pkgArg = aptPackage.Package
			}
			if len(aptPackage.Architectures) > 0 {
				for _, arch := range aptPackage.Architectures {
					pkgArgs = append(pkgArgs, fmt.Sprintf("%s:%s", pkgArg, arch))
				}
			} else {
				pkgArgs = append(pkgArgs, pkgArg)
			}

			for debconfQuestion, debconfSelection := range aptPackage.DebconfSelections {
				commands := []string{
					fmt.Sprintf("set %s %s", debconfQuestion, debconfSelection.Answer),
					fmt.Sprintf("set fset %s %s", debconfQuestion, strconv.FormatBool(debconfSelection.Seen)),
				}
				for _, command := range commands {
					_, err := a.debconfCommunicate(ctx, hst, aptPackage.Package, command)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	cmd := types.Cmd{
		Path: "apt-get",
		Args: []string{"update"},
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

	cmd = types.Cmd{
		Path: "apt-get",
		Args: append([]string{"--yes", "install"}, pkgArgs...),
	}
	waitStatus, stdout, stderr, err = lib.SimpleRun(ctx, hst, cmd)
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
