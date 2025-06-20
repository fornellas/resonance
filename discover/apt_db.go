package discover

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"strings"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/resonance/concurrency"
	"github.com/fornellas/resonance/host/types"
)

type AptDb struct {
	packages          []*AptPackage
	pathToPackagesMap map[string][]*AptPackage
	// name = pkg:architecture
	nameToPackages map[string][]*AptPackage
	pkgToPackages  map[string][]*AptPackage
}

func parseDpkgQueryAdd(
	line string,
	stringSliceValue **[]string,
	aptPackage *AptPackage,
	valuer *func(string) (string, error),
	dpkgDiverts *DpkgDiverts,
) error {
	index := strings.Index(line, "=")
	if index == -1 {
		return fmt.Errorf("unexpected line: %#v", line)
	}
	key := line[:index]
	value := line[index+1:]
	*stringSliceValue = nil
	switch key {
	case "Package":
		aptPackage.pkg = value
	case "Architecture":
		aptPackage.architecture = value
	case "Version":
		aptPackage.version = value
	case "db-fsys:Files":
		aptPackage.dbFsysFiles = []string{}
		*stringSliceValue = &aptPackage.dbFsysFiles
		*valuer = func(line string) (string, error) {
			path := filepath.Clean(line)
			if dpkgDivert := dpkgDiverts.GetDpkgDivert(path); dpkgDivert != nil {
				path = dpkgDivert.DivertTo
			}
			return path, nil
		}
	case "source:Package":
		aptPackage.sourcePackage = value
	case "Conffiles":
		aptPackage.conffiles = []string{}
		*stringSliceValue = &aptPackage.conffiles
		*valuer = func(line string) (string, error) {
			lastSpaceIndex := strings.LastIndex(line, " ")
			if lastSpaceIndex == -1 {
				return "", fmt.Errorf("bug: invalid Conffile: %#v", line)
			}

			path := filepath.Clean(line[:lastSpaceIndex])
			if dpkgDivert := dpkgDiverts.GetDpkgDivert(path); dpkgDivert != nil {
				path = dpkgDivert.DivertTo
			}

			return path, nil
		}
	default:
		return fmt.Errorf("unexpected key: %#v", key)
	}

	return nil
}

func parseDpkgQuery(
	stdoutReader io.Reader,
	dpkgDiverts *DpkgDiverts,
	aptDb *AptDb,
) error {
	scanner := bufio.NewScanner(stdoutReader)

	aptPackage := &AptPackage{}
	var stringSliceValue *[]string = nil
	var valuer func(string) (string, error) = nil
	for scanner.Scan() {
		line := scanner.Text()

		if line == "end" {
			aptDb.packages = append(aptDb.packages, aptPackage)
			aptDb.nameToPackages[aptPackage.Name()] = append(aptDb.nameToPackages[aptPackage.Name()], aptPackage)
			aptDb.pkgToPackages[aptPackage.pkg] = append(aptDb.pkgToPackages[aptPackage.pkg], aptPackage)
			sort.Strings(aptPackage.dbFsysFiles)
			for _, path := range aptPackage.dbFsysFiles {
				aptDb.pathToPackagesMap[path] = append(aptDb.pathToPackagesMap[path], aptPackage)
			}
			sort.Strings(aptPackage.conffiles)

			aptPackage = &AptPackage{}
			stringSliceValue = nil
			continue
		}

		if stringSliceValue == nil || (!strings.HasPrefix(line, " ") && len(line) != 0) {
			if err := parseDpkgQueryAdd(
				line, &stringSliceValue, aptPackage, &valuer, dpkgDiverts,
			); err != nil {
				return err
			}
		} else {
			if len(line) == 0 {
				continue
			}
			value, err := valuer(line[1:])
			if err != nil {
				return err
			}
			*stringSliceValue = append(*stringSliceValue, value)
		}
	}

	sort.SliceStable(aptDb.packages, func(i int, j int) bool {
		return aptDb.packages[i].String() < aptDb.packages[j].String()
	})

	return scanner.Err()
}

func dpkgQuery(
	ctx context.Context,
	host types.Host,
	dpkgDiverts *DpkgDiverts,
	aptDb *AptDb,
) error {
	logger := log.MustLogger(ctx)
	logger.Info("query")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stdoutReader, stdoutWritter, err := os.Pipe()
	if err != nil {
		return err
	}
	stderrBuffer := bytes.Buffer{}

	cmd := types.Cmd{
		Path: "/usr/bin/dpkg-query",
		Args: []string{
			"--show",
			"--showformat=" + strings.Join([]string{
				"Package=${Package}",
				"Architecture=${Architecture}",
				"Version=${Version}",
				"db-fsys:Files=", "${db-fsys:Files}",
				"source:Package=${source:Package}",
				"Conffiles=", "${Conffiles}",
				"end\n",
			}, "\n"),
		},
		Stdout: stdoutWritter,
		Stderr: &stderrBuffer,
	}

	errCh := make(chan error)
	go func() {
		var runErr error
		defer func() {
			errCh <- stdoutWritter.Close()
			errCh <- runErr
			close(errCh)
		}()

		var waitStatus types.WaitStatus
		waitStatus, runErr = host.Run(ctx, cmd)
		if !waitStatus.Success() {
			errCh <- fmt.Errorf("%s:\nstderr:\n%s", cmd, stderrBuffer.String())
			return
		}
	}()

	err = parseDpkgQuery(stdoutReader, dpkgDiverts, aptDb)

	for chErr := range errCh {
		err = errors.Join(err, chErr)
	}

	return err
}

var dpkgVerifyRegexp = regexp.MustCompile(`^(missing  |\?\?([?.5])\?\?\?\?\?\?) ([c ]) (.+)$`)

func dpkgVerifyPackage(
	ctx context.Context,
	host types.Host,
	aptDb *AptDb,
	aptPackage *AptPackage,
) error {
	logger := log.MustLogger(ctx)
	logger.Info(aptPackage.Name())

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stdoutReader, stdoutWritter, err := os.Pipe()
	if err != nil {
		return err
	}
	stderrBuffer := bytes.Buffer{}

	cmd := types.Cmd{
		Path: "/usr/bin/dpkg",
		Args: []string{
			"--verify",
			"--verify-format", "rpm",
			aptPackage.Name(),
		},
		Stdout: stdoutWritter,
		Stderr: &stderrBuffer,
	}

	errCh := make(chan error)
	go func() {
		var runErr error
		defer func() {
			errCh <- stdoutWritter.Close()
			errCh <- runErr
			close(errCh)
		}()

		var waitStatus types.WaitStatus
		waitStatus, runErr = host.Run(ctx, cmd)
		if !waitStatus.Success() {
			errCh <- fmt.Errorf("%s:\nstderr:\n%s", cmd, stderrBuffer.String())
			return
		}
	}()

	scanner := bufio.NewScanner(stdoutReader)
	for scanner.Scan() {
		line := scanner.Text()
		matches := dpkgVerifyRegexp.FindStringSubmatch(line)
		if matches == nil {
			return fmt.Errorf("failed to parse: %#v", line)
		}

		status := matches[1]
		digest := matches[2]
		path := matches[4]

		if status == "missing  " {
			for _, aptPackageOwner := range aptDb.FindOwnerPackages(path) {
				aptPackageOwner.AddMissingPath(path)
			}
		} else {
			if digest == "5" {
				for _, aptPackageOwner := range aptDb.FindOwnerPackages(path) {
					aptPackageOwner.AddDigestCheckFailedPath(path)
				}
			}
		}
	}

	err = nil
	for chErr := range errCh {
		err = errors.Join(err, chErr)
	}

	return err
}

func dpkgVerifyAll(ctx context.Context, host types.Host, aptDb *AptDb) error {
	ctx, _ = log.MustWithGroup(ctx, "dpkg verify")

	concurrencyGroup := concurrency.NewConcurrencyGroup(ctx)

	for _, aptPackage := range aptDb.packages {
		concurrencyGroup.Run(func() error {
			if err := dpkgVerifyPackage(ctx, host, aptDb, aptPackage); err != nil {
				return err
			}
			return nil
		})
	}

	return errors.Join(concurrencyGroup.Wait()...)
}

func getNameFromBinaryPackage(binaryPackage string) string {
	var name string
	nameParts := strings.Split(binaryPackage, ":")
	switch len(nameParts) {
	case 1, 2:
		name = nameParts[0]
	default:
		panic(fmt.Errorf("unexpected binary:Package value: %#v", binaryPackage))
	}
	return name
}

func aptMarkShowmanual(ctx context.Context, host types.Host, aptDb *AptDb) error {
	logger := log.MustLogger(ctx)
	logger.Info("finding manual installed packages")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stdoutReader, stdoutWritter, err := os.Pipe()
	if err != nil {
		return err
	}
	stderrBuffer := bytes.Buffer{}

	cmd := types.Cmd{
		Path:   "/usr/bin/apt-mark",
		Args:   []string{"showmanual"},
		Stdout: stdoutWritter,
		Stderr: &stderrBuffer,
	}

	errCh := make(chan error)
	go func() {
		var runErr error
		defer func() {
			errCh <- stdoutWritter.Close()
			errCh <- runErr
			close(errCh)
		}()

		var waitStatus types.WaitStatus
		waitStatus, runErr = host.Run(ctx, cmd)
		if !waitStatus.Success() {
			errCh <- fmt.Errorf("%s:\nstderr:\n%s", cmd, stderrBuffer.String())
			return
		}
	}()

	scanner := bufio.NewScanner(stdoutReader)
	for scanner.Scan() {
		line := scanner.Text()
		name := getNameFromBinaryPackage(line)
		aptPackages := aptDb.GetPackage(name)
		if aptPackages == nil {
			aptPackages = aptDb.getPackageByPkg(name)
			if aptPackages == nil {
				return fmt.Errorf("can not find package marked as manual by apt: %#v", name)
			}
		}
		for _, aptPackage := range aptPackages {
			aptPackage.MarkManual()
		}
	}

	err = nil
	for chErr := range errCh {
		err = errors.Join(err, chErr)
	}

	return err
}

func aptMarkShowhold(ctx context.Context, host types.Host, aptDb *AptDb) error {
	logger := log.MustLogger(ctx)
	logger.Info("finding held packages")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stdoutReader, stdoutWritter, err := os.Pipe()
	if err != nil {
		return err
	}
	stderrBuffer := bytes.Buffer{}

	cmd := types.Cmd{
		Path:   "/usr/bin/apt-mark",
		Args:   []string{"showhold"},
		Stdout: stdoutWritter,
		Stderr: &stderrBuffer,
	}

	errCh := make(chan error)
	go func() {
		var runErr error
		defer func() {
			errCh <- stdoutWritter.Close()
			errCh <- runErr
			close(errCh)
		}()

		var waitStatus types.WaitStatus
		waitStatus, runErr = host.Run(ctx, cmd)
		if !waitStatus.Success() {
			errCh <- fmt.Errorf("%s:\nstderr:\n%s", cmd, stderrBuffer.String())
			return
		}
	}()

	scanner := bufio.NewScanner(stdoutReader)
	for scanner.Scan() {
		line := scanner.Text()
		name := getNameFromBinaryPackage(line)
		aptPackages := aptDb.GetPackage(name)
		if aptPackages == nil {
			aptPackages = aptDb.getPackageByPkg(name)
			if aptPackages == nil {
				return fmt.Errorf("can not find package marked as manual by apt: %#v", name)
			}
		}
		for _, aptPackage := range aptPackages {
			aptPackage.MarkHold()
		}
	}

	err = nil
	for chErr := range errCh {
		err = errors.Join(err, chErr)
	}

	return err
}

func LoadAptDb(ctx context.Context, host types.Host) (*AptDb, error) {
	ctx, _ = log.MustWithGroup(ctx, "dpkg")

	dpkgDiverts, err := LoadDpkgDiverts(ctx, host)
	if err != nil {
		return nil, err
	}

	aptDb := &AptDb{
		packages:          []*AptPackage{},
		pathToPackagesMap: map[string][]*AptPackage{},
		nameToPackages:    map[string][]*AptPackage{},
		pkgToPackages:     map[string][]*AptPackage{},
	}

	err = dpkgQuery(ctx, host, dpkgDiverts, aptDb)
	if err != nil {
		return nil, err
	}

	if err := dpkgVerifyAll(ctx, host, aptDb); err != nil {
		return nil, err
	}

	if err := aptMarkShowmanual(ctx, host, aptDb); err != nil {
		return nil, err
	}

	if err := aptMarkShowhold(ctx, host, aptDb); err != nil {
		return nil, err
	}

	return aptDb, err
}

func (d *AptDb) FindOwnerPackages(path string) []*AptPackage {
	if aptPackage, ok := d.pathToPackagesMap[path]; ok {
		return aptPackage
	}
	return nil
}

// Get package by AptPackage.pkg
func (d *AptDb) getPackageByPkg(pkg string) []*AptPackage {
	if aptPackages, ok := d.pkgToPackages[pkg]; ok {
		return aptPackages
	}
	return nil
}

// Find packages with given package:architecture name.
func (d *AptDb) GetPackage(name string) []*AptPackage {
	if aptPackages, ok := d.nameToPackages[name]; ok {
		return aptPackages
	}
	return nil
}
