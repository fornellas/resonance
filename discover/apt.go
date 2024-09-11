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
	"sync"

	"strings"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

type AptPackage struct {
	pkg          string
	architecture string
	version      string
	// all package files
	dbFsysFiles   []string
	sourcePackage string
	// config files
	conffiles []string
	// dbFsysFiles that are broken symlinks
	brokenSymLinks []string
	// files which were inferred to be owned by this package
	inferredOwnedPaths []string
	// dbFsysFiles which are missing
	missingPaths []string
	// dbFsysFiles which digest check failed
	digestCheckFailedPaths []string
	// apt marked as manual
	manual bool
	// apt marked as hold
	hold  bool
	mutex sync.Mutex
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

func (d *AptPackage) Name() string {
	return d.pkg + ":" + d.architecture
}

func (d *AptPackage) AddBrokenSymLink(path string) {
	d.mutex.Lock()
	d.brokenSymLinks = append(d.brokenSymLinks, path)
	d.mutex.Unlock()
}

func (d *AptPackage) AddInferredOwnedPath(path string) {
	d.mutex.Lock()
	d.inferredOwnedPaths = append(d.inferredOwnedPaths, path)
	d.mutex.Unlock()
}

func (d *AptPackage) AddMissingPath(path string) {
	d.mutex.Lock()
	d.missingPaths = append(d.missingPaths, path)
	d.mutex.Unlock()
}

func (d *AptPackage) AddDigestCheckFailedPath(path string) {
	d.mutex.Lock()
	d.digestCheckFailedPaths = append(d.digestCheckFailedPaths, path)
	d.mutex.Unlock()
}

func (d *AptPackage) MarkManual() {
	d.mutex.Lock()
	d.manual = true
	d.mutex.Unlock()
}

func (d *AptPackage) MarkHold() {
	d.mutex.Lock()
	d.hold = true
	d.mutex.Unlock()
}

func (d *AptPackage) String() string {
	return d.Name() + "-" + d.version
}

func (d *AptPackage) CompileResources(
	ctx context.Context, host types.Host,
) (string, resourcesPkg.Resources, []string, error) {
	group := d.sourcePackage

	resources := resourcesPkg.Resources{}
	if d.manual || len(d.inferredOwnedPaths) > 0 {
		var version string
		if d.hold {
			version = d.version
		}
		resources = append(resources, &resourcesPkg.APTPackage{
			Package: d.Name(),
			Version: version,
		})
		for _, path := range d.inferredOwnedPaths {
			file := &resourcesPkg.File{
				Path: path,
			}
			if err := file.Load(ctx, host); err != nil {
				return "", nil, nil, err
			}
			resources = append(resources, file)
		}
	}

	issues := []string{}
	for _, path := range d.brokenSymLinks {
		issues = append(issues, fmt.Sprintf("broken symlink: %s", path))
	}
	for _, path := range d.missingPaths {
		issues = append(issues, fmt.Sprintf("missing: %s", path))
	}
	for _, path := range d.digestCheckFailedPaths {
		issues = append(issues, fmt.Sprintf("digest check failed: %s", path))
	}

	return group, resources, issues, nil
}

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

func dpkgVerify(ctx context.Context, host types.Host, aptDb *AptDb) error {
	logger := log.MustLogger(ctx)
	logger.Info("verifying")

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
			for _, aptPackage := range aptDb.FindOwnerPackages(path) {
				aptPackage.AddMissingPath(path)
			}
		} else {
			if digest == "5" {
				for _, aptPackage := range aptDb.FindOwnerPackages(path) {
					aptPackage.AddDigestCheckFailedPath(path)
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
	ctx, _ = log.MustContextLoggerWithSection(ctx, "dpkg")

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

	if err := dpkgVerify(ctx, host, aptDb); err != nil {
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
