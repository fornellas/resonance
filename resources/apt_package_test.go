package resources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/slogxt/log"
)

func TestAPTPackage(t *testing.T) {
	t.Run("Satisfies()", func(t *testing.T) {
		tests := []struct {
			name     string
			current  *APTPackage
			target   *APTPackage
			expected bool
		}{
			// Package
			{
				name: "same package, no extra fields",
				current: &APTPackage{
					Package: "wget",
				},
				target: &APTPackage{
					Package: "wget",
				},
				expected: true,
			},
			{
				name: "different package names",
				current: &APTPackage{
					Package: "wget",
				},
				target: &APTPackage{
					Package: "curl",
				},
				expected: false,
			},
			{
				name: "current has architectures, target doesn't - should satisfy",
				current: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
				},
				target: &APTPackage{
					Package: "wget",
				},
				expected: true,
			},
			// Absent
			{
				name: "absent mismatch",
				current: &APTPackage{
					Package: "wget",
					Absent:  false,
				},
				target: &APTPackage{
					Package: "wget",
					Absent:  true,
				},
				expected: false,
			},
			{
				name: "absent match",
				current: &APTPackage{
					Package: "wget",
					Absent:  true,
				},
				target: &APTPackage{
					Package: "wget",
					Absent:  true,
				},
				expected: true,
			},
			// Architectures
			{
				name: "current has all required architectures",
				current: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64", "i386"},
				},
				target: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
				},
				expected: true,
			},
			{
				name: "current has architectures, target has other arch - should not satisfy",
				current: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
				},
				target: &APTPackage{
					Package:       "wget",
					Architectures: []string{"i386"},
				},
				expected: false,
			},
			{
				name: "target has architectures, current doesn't - should not satisfy",
				current: &APTPackage{
					Package: "wget",
				},
				target: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
				},
				expected: false,
			},

			{
				name: "current missing required architecture",
				current: &APTPackage{
					Package:       "wget",
					Architectures: []string{"i386"},
				},
				target: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
				},
				expected: false,
			},
			// Version
			{
				name: "current has version, target doesn't - should satisfy",
				current: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1ubuntu4.1",
				},
				target: &APTPackage{
					Package: "wget",
				},
				expected: true,
			},
			{
				name: "target has version, current doesn't - should not satisfy",
				current: &APTPackage{
					Package: "wget",
				},
				target: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1ubuntu4.1",
				},
				expected: false,
			},
			{
				name: "matching versions",
				current: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1ubuntu4.1",
				},
				target: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1ubuntu4.1",
				},
				expected: true,
			},
			{
				name: "different versions",
				current: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1ubuntu4.1",
				},
				target: &APTPackage{
					Package: "wget",
					Version: "1.21.3-1ubuntu4.1",
				},
				expected: false,
			},
			// DebconfSelections
			{
				name: "target has debconf selections, current doesn't",
				current: &APTPackage{
					Package: "wget",
				},
				target: &APTPackage{
					Package: "wget",
					DebconfSelections: map[DebconfQuestion]DebconfSelection{
						"wget/question": {Answer: "yes", Seen: true},
					},
				},
				expected: false,
			},
			{
				name: "current has debconf selections, target doesn't",
				current: &APTPackage{
					Package: "wget",
					DebconfSelections: map[DebconfQuestion]DebconfSelection{
						"wget/question": {Answer: "yes", Seen: true},
					},
				},
				target: &APTPackage{
					Package: "wget",
				},
				expected: true,
			},
			{
				name: "matching debconf selections",
				current: &APTPackage{
					Package: "wget",
					DebconfSelections: map[DebconfQuestion]DebconfSelection{
						"wget/question": {Answer: "yes", Seen: true},
					},
				},
				target: &APTPackage{
					Package: "wget",
					DebconfSelections: map[DebconfQuestion]DebconfSelection{
						"wget/question": {Answer: "yes", Seen: true},
					},
				},
				expected: true,
			},
			{
				name: "different debconf answers",
				current: &APTPackage{
					Package: "wget",
					DebconfSelections: map[DebconfQuestion]DebconfSelection{
						"wget/question": {Answer: "yes", Seen: true},
					},
				},
				target: &APTPackage{
					Package: "wget",
					DebconfSelections: map[DebconfQuestion]DebconfSelection{
						"wget/question": {Answer: "no", Seen: true},
					},
				},
				expected: false,
			},
			// Misc
			{
				name: "current has both architectures and version, target doesn't - should satisfy",
				current: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
					Version:       "1.21.4-1ubuntu4.1",
				},
				target: &APTPackage{
					Package: "wget",
				},
				expected: true,
			},
			// Hold
			{
				name: "hold mismatch - current held, target not",
				current: &APTPackage{
					Package: "wget",
					Hold:    true,
				},
				target: &APTPackage{
					Package: "wget",
					Hold:    false,
				},
				expected: false,
			},
			{
				name: "hold mismatch - current not held, target held",
				current: &APTPackage{
					Package: "wget",
					Hold:    false,
				},
				target: &APTPackage{
					Package: "wget",
					Hold:    true,
				},
				expected: false,
			},
			{
				name: "hold match - both held",
				current: &APTPackage{
					Package: "wget",
					Hold:    true,
				},
				target: &APTPackage{
					Package: "wget",
					Hold:    true,
				},
				expected: true,
			},
			{
				name: "hold match - both not held",
				current: &APTPackage{
					Package: "wget",
					Hold:    false,
				},
				target: &APTPackage{
					Package: "wget",
					Hold:    false,
				},
				expected: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.current.Satisfies(tt.target)
				if result != tt.expected {
					t.Errorf("APTPackage.Satisfies() = %v, want %v", result, tt.expected)
					t.Errorf("Current: %+v", tt.current)
					t.Errorf("Target: %+v", tt.target)
				}
			})
		}
	})

	t.Run("Validate()", func(t *testing.T) {
		tests := []struct {
			name        string
			aptPackage  *APTPackage
			expectError bool
			errorMsg    string
		}{
			{
				name: "valid package with all fields",
				aptPackage: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64", "i386"},
					Version:       "1.21.4-1ubuntu4.1",
				},
				expectError: false,
			},
			{
				name: "valid package minimal",
				aptPackage: &APTPackage{
					Package: "curl",
				},
				expectError: false,
			},
			{
				name: "valid package with epoch",
				aptPackage: &APTPackage{
					Package: "libc6",
					Version: "2:2.31-0ubuntu9.9",
				},
				expectError: false,
			},
			{
				name: "valid package with tilde",
				aptPackage: &APTPackage{
					Package: "test",
					Version: "1.0~beta1",
				},
				expectError: false,
			},
			{
				name: "valid package with plus in name",
				aptPackage: &APTPackage{
					Package: "g++",
				},
				expectError: false,
			},
			{
				name: "valid package with dots and dashes",
				aptPackage: &APTPackage{
					Package: "lib.test-package",
				},
				expectError: false,
			},
			{
				name: "empty package name",
				aptPackage: &APTPackage{
					Package: "",
				},
				expectError: true,
				errorMsg:    "invalid package",
			},
			{
				name: "package name starting with invalid character",
				aptPackage: &APTPackage{
					Package: "-invalid",
				},
				expectError: true,
				errorMsg:    "invalid package",
			},
			{
				name: "package name with uppercase",
				aptPackage: &APTPackage{
					Package: "Invalid",
				},
				expectError: true,
				errorMsg:    "invalid package",
			},
			{
				name: "package name with invalid characters",
				aptPackage: &APTPackage{
					Package: "pack@ge",
				},
				expectError: true,
				errorMsg:    "invalid package",
			},
			{
				name: "package name too short",
				aptPackage: &APTPackage{
					Package: "a",
				},
				expectError: true,
				errorMsg:    "invalid package",
			},
			{
				name: "package name starting with dot",
				aptPackage: &APTPackage{
					Package: ".invalid",
				},
				expectError: true,
				errorMsg:    "invalid package",
			},
			{
				name: "valid architecture",
				aptPackage: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
				},
				expectError: false,
			},
			{
				name: "invalid architecture with uppercase",
				aptPackage: &APTPackage{
					Package:       "wget",
					Architectures: []string{"AMD64"},
				},
				expectError: true,
				errorMsg:    "invalid package",
			},
			{
				name: "invalid architecture with underscore",
				aptPackage: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd_64"},
				},
				expectError: true,
				errorMsg:    "invalid package",
			},
			{
				name: "invalid architecture empty",
				aptPackage: &APTPackage{
					Package:       "wget",
					Architectures: []string{""},
				},
				expectError: true,
				errorMsg:    "invalid package",
			},
			{
				name: "invalid architecture with dots",
				aptPackage: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd.64"},
				},
				expectError: true,
				errorMsg:    "invalid package",
			},
			{
				name: "version ending with plus",
				aptPackage: &APTPackage{
					Package: "wget",
					Version: "1.21.4+",
				},
				expectError: true,
				errorMsg:    "version` can't end in +",
			},
			{
				name: "version ending with minus",
				aptPackage: &APTPackage{
					Package: "wget",
					Version: "1.21.4-",
				},
				expectError: true,
				errorMsg:    "version` can't end in -",
			},
			{
				name: "invalid version format",
				aptPackage: &APTPackage{
					Package: "wget",
					Version: "invalid-version-format!",
				},
				expectError: true,
				errorMsg:    "invalid version",
			},
			{
				name: "version starting with letter",
				aptPackage: &APTPackage{
					Package: "wget",
					Version: "abc123",
				},
				expectError: true,
				errorMsg:    "invalid version",
			},
			{
				name: "empty version is valid",
				aptPackage: &APTPackage{
					Package: "wget",
					Version: "",
				},
				expectError: false,
			},
			{
				name: "valid complex version",
				aptPackage: &APTPackage{
					Package: "complex-package.name",
					Version: "1:2.3.4~rc1+dfsg-5ubuntu2.1",
				},
				expectError: false,
			},
			{
				name: "valid version with just numbers",
				aptPackage: &APTPackage{
					Package: "simple",
					Version: "123",
				},
				expectError: false,
			},
			{
				name: "valid version with native package format",
				aptPackage: &APTPackage{
					Package: "native",
					Version: "1.2.3",
				},
				expectError: false,
			},
			{
				name: "multiple valid architectures",
				aptPackage: &APTPackage{
					Package:       "multiarch",
					Architectures: []string{"amd64", "i386", "arm64", "armhf"},
				},
				expectError: false,
			},
			{
				name: "one invalid architecture among valid ones",
				aptPackage: &APTPackage{
					Package:       "multiarch",
					Architectures: []string{"amd64", "Invalid", "arm64"},
				},
				expectError: true,
				errorMsg:    "invalid package",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.aptPackage.Validate()
				if tt.expectError {
					require.Error(t, err)
					require.Contains(t, err.Error(), tt.errorMsg)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}

func TestAPTPackages(t *testing.T) {
	t.Run("Apply()", func(t *testing.T) {
		dockerHost, _ := host.GetTestDockerHost(t, "debian")

		ctx := log.WithTestLogger(t.Context())

		agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
		require.NoError(t, err)

		aptPackages := &APTPackages{}
		packages := []*APTPackage{
			{
				Package: "nano",
			},
		}
		err = aptPackages.Apply(ctx, agentHost, packages)
		require.NoError(t, err)

		cmd := types.Cmd{
			Path: "/usr/bin/dpkg",
			Args: []string{"-l", "nano"},
		}
		waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, agentHost, cmd)
		require.NoError(t, err)
		require.True(t, waitStatus.Success(), "dpkg -l nano failed: %s\nSTDOUT:\n%s\nSTDERR:\n%s", waitStatus.String(), stdout, stderr)
		require.True(t, strings.Contains(stdout, "nano"), "nano package not found in dpkg output: %s", stdout)
	})
}
