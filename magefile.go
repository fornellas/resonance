// +build mage

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
)

var (
	// Paths and environment variables
	cachePath    = getCachePath()
	goModVersion = getGoModVersion()
	goModule     = getGoModule()
	goos         = runtime.GOOS
	goarch       = runtime.GOARCH
)

// GetCachePath determines the cache path depending on the OS
func getCachePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	switch goos {
	case "linux":
		return filepath.Join(homeDir, ".cache", "resonance")
	case "darwin":
		return filepath.Join(homeDir, "Library", "Caches", "resonance")
	default:
		log.Fatalf("Unsupported system: %s", goos)
		return ""
	}
}

// GetGoModVersion extracts the Go version from go.mod
func getGoModVersion() string {
	out, err := exec.Command("awk", "/^go /{print $2}", "go.mod").Output()
	if err != nil {
		log.Fatal("Error retrieving Go version from go.mod:", err)
	}
	return strings.TrimSpace(string(out))
}

// GetGoModule extracts the Go module name from go.mod
func getGoModule() string {
	out, err := exec.Command("awk", "/^module /{print $2}", "go.mod").Output()
	if err != nil {
		log.Fatal("Error retrieving Go module from go.mod:", err)
	}
	return strings.TrimSpace(string(out))
}

// Clean removes build and cache files
func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll(filepath.Join(cachePath, "GOROOT"))
	os.RemoveAll(filepath.Join(cachePath, "GOCACHE"))
	os.RemoveAll(filepath.Join(cachePath, "GOMODCACHE"))
	fmt.Println("Clean complete")
}

// Go fetches and installs Go if it's not already installed
func Go() error {
	goroot := filepath.Join(cachePath, "GOROOT", fmt.Sprintf("go%s.%s-%s", goModVersion, goos, goarch))
	if _, err := os.Stat(filepath.Join(goroot, "bin", "go")); os.IsNotExist(err) {
		fmt.Println("Downloading and installing Go...")
		url := fmt.Sprintf("https://go.dev/dl/go%s.%s-%s.tar.gz", goModVersion, goos, goarch)
		cmd := exec.Command("curl", "-sSfL", url, "|", "tar", "-zx", "-C", filepath.Join(cachePath, "GOROOT"))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install Go: %w", err)
		}
		fmt.Println("Go installation complete")
	}
	return nil
}

// InstallProtoc installs protobuf compiler
func InstallProtoc() error {
	protocVersion := "28.0"
	protocOS := goos
	protocArch := goarch
	protocBinPath := filepath.Join(cachePath, "protoc", protocVersion, protocOS+"-"+protocArch)

	if _, err := os.Stat(filepath.Join(protocBinPath, "protoc")); os.IsNotExist(err) {
		fmt.Println("Installing Protoc...")
		url := fmt.Sprintf("https://github.com/protocolbuffers/protobuf/releases/download/v%s/protoc-%s-%s-%s.zip", protocVersion, protocVersion, protocOS, protocArch)
		cmd := exec.Command("curl", "-sSfL", url, "-o", filepath.Join(cachePath, "protoc.zip"))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to download protoc: %w", err)
		}
		cmd = exec.Command("unzip", "-p", filepath.Join(cachePath, "protoc.zip"), "bin/protoc", ">", filepath.Join(protocBinPath, "protoc.tmp"))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to extract protoc: %w", err)
		}
		fmt.Println("Protoc installation complete")
	}
	return nil
}

// Lint runs all linters
func Lint() error {
	fmt.Println("Running linters...")
	// Run goimports
	cmd := exec.Command("go", "run", "golang.org/x/tools/cmd/goimports", "-w", "-local", goModule, ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("goimports failed: %w", err)
	}
	// Add other linters here...
	return nil
}

// GoTest runs the tests
func GoTest() error {
	fmt.Println("Running tests...")
	cmd := exec.Command("go", "test", "./...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go test failed: %w", err)
	}
	return nil
}

// CI runs the full CI pipeline: lint, test, build
func CI() {
	mg.Deps(Lint, GoTest)
}

// Help prints help information
func Help() {
	fmt.Println(`
Available targets:
  clean             Cleans up build artifacts and cache
  go                Installs Go if needed
  install-protoc    Installs Protoc
  lint              Runs all linters
  test              Runs all tests
  ci                Runs lint, test, and build steps for CI
`)
}
