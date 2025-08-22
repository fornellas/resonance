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
	Answer string
	Seen   bool
}

// APTPackage manages APT packages.
type APTPackage struct {
	// The name of the package
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#package
	Package string
	// Whether to remove the package
	Absent bool
	// Architectures.
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#architecture
	Architectures []string
	// Package version.
	// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	Version string
	// Whether the package should be held to prevent automatic upgrades
	Hold bool
	// Package debconf selections.
	// See https://wiki.debian.org/debconf
	DebconfSelections map[DebconfQuestion]DebconfSelection
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

	// Hold logic validation
	if a.Absent && a.Hold {
		return fmt.Errorf("hold can't be set when package is absent")
	}
	if !a.Absent {
		if a.Version == "" && a.Hold {
			return fmt.Errorf("hold can't be set when version is unset")
		}
		if a.Version != "" && !a.Hold {
			return fmt.Errorf("hold must be set when version is set")
		}
	}

	return nil
}

func (a *APTPackage) Satisfies(b *APTPackage) bool {
	if a.Package != b.Package {
		return false
	}

	if a.Absent != b.Absent {
		return false
	}

	if len(b.Architectures) > 0 {
		for _, arch := range b.Architectures {
			if !slices.Contains(a.Architectures, arch) {
				return false
			}
		}
	}

	if len(b.Version) > 0 {
		if a.Version != b.Version {
			return false
		}
	}

	if a.Hold != b.Hold {
		return false
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
		aptPackage.Hold = false
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
		return "", fmt.Errorf("%s failed: %s\nSTDIN:\n%s\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), command, stdout, stderr)
	}
	if len(stderr) > 0 {
		return "", fmt.Errorf("%s failed: %s\nSTDIN:\n%s\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), command, stdout, stderr)
	}
	var value string
	if strings.HasPrefix(stdout, "0 ") {
		value = strings.TrimSuffix(stdout[2:], "\n")
	} else {
		if stdout != "0" {
			return "", fmt.Errorf("%s failed: %s\nSTDIN:\n%s\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), command, stdout, stderr)
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
				Env:  []string{"LANG=C"},
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

func (a *APTPackages) loadHoldStatus(ctx context.Context, hst types.Host, aptPackages []*APTPackage) error {
	cmd := types.Cmd{
		Path: "/usr/bin/dpkg",
		Args: []string{"--get-selections"},
	}
	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, hst, cmd)
	if err != nil {
		return fmt.Errorf("failed to run dpkg --get-selections: %w\n%s\nSTDOUT:\n%s\nSTDERR:\n%s", err, cmd, stdout, stderr)
	}
	if !waitStatus.Success() {
		return fmt.Errorf("dpkg --get-selections failed: %s\nSTDOUT:\n%s\nSTDERR:\n%s", waitStatus.String(), stdout, stderr)
	}

	heldPackages := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == "hold" {
			heldPackages[parts[0]] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed scanning dpkg --get-selections output: %w", err)
	}

	for _, aptPackage := range aptPackages {
		if !aptPackage.Absent {
			aptPackage.Hold = heldPackages[aptPackage.Package]
		}
	}

	return nil
}

func (a *APTPackages) Load(ctx context.Context, hst types.Host, aptPackages []*APTPackage) error {
	packageQueries, packageToResource := a.preparePackageQueries(aptPackages)

	stdout, err := a.runDpkgQuery(ctx, hst, packageQueries, len(aptPackages))
	if err != nil {
		return err
	}

	if err := a.processDpkgOutput(stdout, packageToResource); err != nil {
		return fmt.Errorf("failed scanning dpkg-query output: %w", err)
	}

	if err := a.loadDebconfSelections(ctx, hst, aptPackages); err != nil {
		return fmt.Errorf("failed loading debconf: %w", err)
	}

	if err := a.loadHoldStatus(ctx, hst, aptPackages); err != nil {
		return fmt.Errorf("failed loading hold status: %w", err)
	}

	return nil
}

func (a *APTPackages) Resolve(ctx context.Context, hst types.Host, aptPackages []*APTPackage) error {
	return nil
}

func (a *APTPackages) applyHolds(ctx context.Context, hst types.Host, aptPackages []*APTPackage) error {
	var selections strings.Builder

	for _, aptPackage := range aptPackages {
		if aptPackage.Absent {
			continue
		}

		status := "install"
		if aptPackage.Hold {
			status = "hold"
		}

		if len(aptPackage.Architectures) > 0 {
			for _, arch := range aptPackage.Architectures {
				selections.WriteString(fmt.Sprintf("%s:%s %s\n", aptPackage.Package, arch, status))
			}
		} else {
			selections.WriteString(fmt.Sprintf("%s %s\n", aptPackage.Package, status))
		}
	}

	if selections.Len() == 0 {
		return nil
	}

	stdinReader := strings.NewReader(selections.String())
	stdoutBuffer := bytes.Buffer{}
	stderrBuffer := bytes.Buffer{}
	cmd := types.Cmd{
		Path:   "/usr/bin/dpkg",
		Args:   []string{"--set-selections"},
		Stdin:  stdinReader,
		Stdout: &stdoutBuffer,
		Stderr: &stderrBuffer,
	}

	waitStatus, err := hst.Run(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to run dpkg --set-selections: %w", err)
	}

	stdout := stdoutBuffer.String()
	stderr := stderrBuffer.String()

	if !waitStatus.Success() {
		return fmt.Errorf("dpkg --set-selections failed: %s\nSTDOUT:\n%s\nSTDERR:\n%s", waitStatus.String(), stdout, stderr)
	}

	return nil
}

func (a *APTPackages) loadCurrentArchitecturesForPackages(ctx context.Context, hst types.Host, aptPackages []*APTPackage) (map[string][]string, error) {
	packagesNeedingArchCheck := make([]*APTPackage, 0)
	for _, aptPackage := range aptPackages {
		if !aptPackage.Absent && len(aptPackage.Architectures) > 0 {
			packagesNeedingArchCheck = append(packagesNeedingArchCheck, &APTPackage{
				Package: aptPackage.Package,
			})
		}
	}

	if len(packagesNeedingArchCheck) == 0 {
		return make(map[string][]string), nil
	}

	if err := a.Load(ctx, hst, packagesNeedingArchCheck); err != nil {
		return nil, fmt.Errorf("failed to load current state for architecture checking: %w", err)
	}

	currentArchs := make(map[string][]string)
	for _, pkg := range packagesNeedingArchCheck {
		if !pkg.Absent {
			currentArchs[pkg.Package] = pkg.Architectures
		}
	}

	return currentArchs, nil
}

func (a *APTPackages) buildPackageArguments(aptPackages []*APTPackage, currentArchs map[string][]string) []string {
	pkgArgs := []string{}
	for _, aptPackage := range aptPackages {
		if aptPackage.Absent {
			pkgArgs = append(pkgArgs, fmt.Sprintf("%s-", aptPackage.Package))
		} else {
			pkgArgs = append(pkgArgs, a.buildInstallArguments(aptPackage, currentArchs)...)
		}
	}
	return pkgArgs
}

func (a *APTPackages) buildInstallArguments(aptPackage *APTPackage, currentArchs map[string][]string) []string {
	var pkgArg string
	if len(aptPackage.Version) > 0 {
		pkgArg = fmt.Sprintf("%s=%s", aptPackage.Package, aptPackage.Version)
	} else {
		pkgArg = aptPackage.Package
	}

	if len(aptPackage.Architectures) > 0 {
		return a.buildArchitectureArguments(aptPackage, pkgArg, currentArchs)
	}

	return []string{pkgArg}
}

func (a *APTPackages) buildArchitectureArguments(aptPackage *APTPackage, pkgArg string, currentArchs map[string][]string) []string {
	args := []string{}

	// Install desired architectures
	for _, arch := range aptPackage.Architectures {
		args = append(args, fmt.Sprintf("%s:%s", pkgArg, arch))
	}

	// Remove unwanted architectures that are currently installed
	if currentArch, exists := currentArchs[aptPackage.Package]; exists {
		for _, arch := range currentArch {
			if !slices.Contains(aptPackage.Architectures, arch) {
				args = append(args, fmt.Sprintf("%s:%s-", aptPackage.Package, arch))
			}
		}
	}

	return args
}

func (a *APTPackages) configureDebconfSelections(ctx context.Context, hst types.Host, aptPackages []*APTPackage) error {
	for _, aptPackage := range aptPackages {
		if aptPackage.Absent {
			continue
		}

		for debconfQuestion, debconfSelection := range aptPackage.DebconfSelections {
			commands := []string{
				fmt.Sprintf("set %s %s", debconfQuestion, debconfSelection.Answer),
				fmt.Sprintf("fset %s seen %s", debconfQuestion, strconv.FormatBool(debconfSelection.Seen)),
			}
			for _, command := range commands {
				_, err := a.debconfCommunicate(ctx, hst, aptPackage.Package, command)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (a *APTPackages) runAptCommands(ctx context.Context, hst types.Host, pkgArgs []string) error {
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

func (a *APTPackages) Apply(ctx context.Context, hst types.Host, aptPackages []*APTPackage) error {
	currentArchs, err := a.loadCurrentArchitecturesForPackages(ctx, hst, aptPackages)
	if err != nil {
		return err
	}

	pkgArgs := a.buildPackageArguments(aptPackages, currentArchs)

	if err := a.configureDebconfSelections(ctx, hst, aptPackages); err != nil {
		return err
	}

	if err := a.runAptCommands(ctx, hst, pkgArgs); err != nil {
		return err
	}

	if err := a.applyHolds(ctx, hst, aptPackages); err != nil {
		return fmt.Errorf("failed applying holds: %w", err)
	}

	return nil
}
