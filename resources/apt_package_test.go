package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
)

func TestAPTPackage(t *testing.T) {
	t.Run("Satisfies()", func(t *testing.T) {
		tests := []struct {
			name     string
			a        *APTPackage
			b        *APTPackage
			expected bool
		}{
			// Package
			{
				name: "same package, no extra fields",
				a: &APTPackage{
					Package: "wget",
				},
				b: &APTPackage{
					Package: "wget",
				},
				expected: true,
			},
			{
				name: "different package names",
				a: &APTPackage{
					Package: "wget",
				},
				b: &APTPackage{
					Package: "curl",
				},
				expected: false,
			},
			{
				name: "a has architectures, b doesn't - should satisfy",
				a: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
				},
				b: &APTPackage{
					Package: "wget",
				},
				expected: true,
			},
			// Absent
			{
				name: "absent mismatch",
				a: &APTPackage{
					Package: "wget",
					Absent:  false,
				},
				b: &APTPackage{
					Package: "wget",
					Absent:  true,
				},
				expected: false,
			},
			{
				name: "absent match",
				a: &APTPackage{
					Package: "wget",
					Absent:  true,
				},
				b: &APTPackage{
					Package: "wget",
					Absent:  true,
				},
				expected: true,
			},
			// Architectures
			{
				name: "a has all required architectures",
				a: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64", "i386"},
				},
				b: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
				},
				expected: true,
			},
			{
				name: "a has architectures, b has other arch - should not satisfy",
				a: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
				},
				b: &APTPackage{
					Package:       "wget",
					Architectures: []string{"i386"},
				},
				expected: false,
			},
			{
				name: "a has architectures, b doesn't - should not satisfy",
				a: &APTPackage{
					Package: "wget",
				},
				b: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
				},
				expected: false,
			},

			{
				name: "a missing required architecture",
				a: &APTPackage{
					Package:       "wget",
					Architectures: []string{"i386"},
				},
				b: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
				},
				expected: false,
			},
			// Version
			{
				name: "a has version, b doesn't",
				a: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1ubuntu4.1",
					Hold:    true,
				},
				b: &APTPackage{
					Package: "wget",
				},
				expected: true,
			},
			{
				name: "a has version, b doesn't - should not satisfy",
				a: &APTPackage{
					Package: "wget",
				},
				b: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1ubuntu4.1",
					Hold:    true,
				},
				expected: false,
			},
			{
				name: "matching versions",
				a: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1ubuntu4.1",
					Hold:    true,
				},
				b: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1ubuntu4.1",
					Hold:    true,
				},
				expected: true,
			},
			{
				name: "different versions",
				a: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1ubuntu4.1",
					Hold:    true,
				},
				b: &APTPackage{
					Package: "wget",
					Version: "1.21.3-1ubuntu4.1",
					Hold:    true,
				},
				expected: false,
			},
			// DebconfSelections
			{
				name: "b has debconf selections, a doesn't",
				a: &APTPackage{
					Package: "wget",
				},
				b: &APTPackage{
					Package: "wget",
					DebconfSelections: map[DebconfQuestion]DebconfAnswer{
						"wget/question": "yes",
					},
				},
				expected: false,
			},
			{
				name: "a has debconf selections, b doesn't",
				a: &APTPackage{
					Package: "wget",
					DebconfSelections: map[DebconfQuestion]DebconfAnswer{
						"wget/question": "yes",
					},
				},
				b: &APTPackage{
					Package: "wget",
				},
				expected: true,
			},
			{
				name: "matching debconf selections",
				a: &APTPackage{
					Package: "wget",
					DebconfSelections: map[DebconfQuestion]DebconfAnswer{
						"wget/question": "yes",
					},
				},
				b: &APTPackage{
					Package: "wget",
					DebconfSelections: map[DebconfQuestion]DebconfAnswer{
						"wget/question": "yes",
					},
				},
				expected: true,
			},
			{
				name: "different debconf answers",
				a: &APTPackage{
					Package: "wget",
					DebconfSelections: map[DebconfQuestion]DebconfAnswer{
						"wget/question": "yes",
					},
				},
				b: &APTPackage{
					Package: "wget",
					DebconfSelections: map[DebconfQuestion]DebconfAnswer{
						"wget/question": "no",
					},
				},
				expected: false,
			},
			// Misc
			{
				name: "a has both architectures and version, b doesn't",
				a: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd64"},
					Version:       "1.21.4-1ubuntu4.1",
					Hold:          true,
				},
				b: &APTPackage{
					Package: "wget",
				},
				expected: true,
			},
			// Hold
			{
				name: "hold mismatch - a held, b not",
				a: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1",
					Hold:    true,
				},
				b: &APTPackage{
					Package: "wget",
					Hold:    false,
				},
				expected: true,
			},
			{
				name: "hold mismatch - a not held, b held",
				a: &APTPackage{
					Package: "wget",
					Hold:    false,
				},
				b: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1",
					Hold:    true,
				},
				expected: false,
			},
			{
				name: "hold match - both held",
				a: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1",
					Hold:    true,
				},
				b: &APTPackage{
					Package: "wget",
					Version: "1.21.4-1",
					Hold:    true,
				},
				expected: true,
			},
			{
				name: "hold match - both not held",
				a: &APTPackage{
					Package: "wget",
					Hold:    false,
				},
				b: &APTPackage{
					Package: "wget",
					Hold:    false,
				},
				expected: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.a.Satisfies(tt.b)
				if result != tt.expected {
					t.Errorf("APTPackage.Satisfies() = %v, want %v", result, tt.expected)
					t.Errorf("a: %+v", tt.a)
					t.Errorf("b: %+v", tt.b)
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
					Hold:          true,
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
					Hold:    true,
				},
				expectError: false,
			},
			{
				name: "valid package with tilde",
				aptPackage: &APTPackage{
					Package: "test",
					Version: "1.0~beta1",
					Hold:    true,
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
				errorMsg:    "invalid architecture",
			},
			{
				name: "invalid architecture with underscore",
				aptPackage: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd_64"},
				},
				expectError: true,
				errorMsg:    "invalid architecture",
			},
			{
				name: "invalid architecture empty",
				aptPackage: &APTPackage{
					Package:       "wget",
					Architectures: []string{""},
				},
				expectError: true,
				errorMsg:    "invalid architecture",
			},
			{
				name: "invalid architecture with dots",
				aptPackage: &APTPackage{
					Package:       "wget",
					Architectures: []string{"amd.64"},
				},
				expectError: true,
				errorMsg:    "invalid architecture",
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
					Hold:    true,
				},
				expectError: false,
			},
			{
				name: "valid version with just numbers",
				aptPackage: &APTPackage{
					Package: "simple",
					Version: "123",
					Hold:    true,
				},
				expectError: false,
			},
			{
				name: "valid version with native package format",
				aptPackage: &APTPackage{
					Package: "native",
					Version: "1.2.3",
					Hold:    true,
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
				errorMsg:    "invalid architecture",
			},
			// Hold logic validation tests
			{
				name: "absent package with hold set - should fail",
				aptPackage: &APTPackage{
					Package: "wget",
					Absent:  true,
					Hold:    true,
				},
				expectError: true,
				errorMsg:    "hold can't be set when package is absent",
			},
			{
				name: "absent package without hold - should pass",
				aptPackage: &APTPackage{
					Package: "wget",
					Absent:  true,
					Hold:    false,
				},
				expectError: false,
			},
			{
				name: "non-absent package with version unset and hold set - should fail",
				aptPackage: &APTPackage{
					Package: "wget",
					Absent:  false,
					Version: "",
					Hold:    true,
				},
				expectError: true,
				errorMsg:    "hold can't be set when version is unset",
			},
			{
				name: "non-absent package with version unset and hold unset - should pass",
				aptPackage: &APTPackage{
					Package: "wget",
					Absent:  false,
					Version: "",
					Hold:    false,
				},
				expectError: false,
			},
			{
				name: "non-absent package with version set and hold unset - should fail",
				aptPackage: &APTPackage{
					Package: "wget",
					Absent:  false,
					Version: "1.21.4-1ubuntu4.1",
					Hold:    false,
				},
				expectError: true,
				errorMsg:    "hold must be set when version is set",
			},
			{
				name: "non-absent package with version set and hold set - should pass",
				aptPackage: &APTPackage{
					Package: "wget",
					Absent:  false,
					Version: "1.21.4-1ubuntu4.1",
					Hold:    true,
				},
				expectError: false,
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

// runAndRequireSuccess runs a command and requires it to succeed, providing detailed error info on failure
func runAndRequireSuccess(t *testing.T, ctx context.Context, host types.BaseHost, cmd types.Cmd) string {
	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, host, cmd)
	require.NoError(t, err)
	require.True(t, waitStatus.Success(), "Command %s %v failed: %s\nSTDOUT:\n%s\nSTDERR:\n%s", cmd.Path, cmd.Args, waitStatus.String(), stdout, stderr)
	return stdout
}

func TestAPTPackages(t *testing.T) {
	t.Run("Apply()", func(t *testing.T) {
		for _, image := range []string{
			// "debian:bookworm",
			// "debian:trixie",
			// "ubuntu:22.04",
			"ubuntu:24.04",
		} {
			t.Run(image, func(t *testing.T) {
				t.Run("basic package installation", func(t *testing.T) {
					t.Parallel()

					dockerHost, _ := host.GetTestDockerHost(t, image)
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
					stdout := runAndRequireSuccess(t, ctx, agentHost, cmd)
					require.True(t, strings.Contains(stdout, "nano"), "nano package not found in dpkg output: %s", stdout)
				})

				t.Run("package removal", func(t *testing.T) {
					t.Parallel()

					dockerHost, _ := host.GetTestDockerHost(t, image)
					ctx := log.WithTestLogger(t.Context())
					agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
					require.NoError(t, err)

					aptPackages := &APTPackages{}

					// First install nano
					packages := []*APTPackage{
						{
							Package: "nano",
						},
					}
					err = aptPackages.Apply(ctx, agentHost, packages)
					require.NoError(t, err)

					// Then remove it
					packages = []*APTPackage{
						{
							Package: "nano",
							Absent:  true,
						},
					}
					err = aptPackages.Apply(ctx, agentHost, packages)
					require.NoError(t, err)

					// Verify it's removed
					cmd := types.Cmd{
						Path: "/usr/bin/dpkg",
						Args: []string{"-l", "nano"},
					}
					waitStatus, stdout, _, err := lib.SimpleRun(ctx, agentHost, cmd)
					require.NoError(t, err)
					require.False(t, waitStatus.Success() && strings.Contains(stdout, "ii  nano"), "nano package should be removed but found in dpkg output: %s", stdout)
				})

				t.Run("package with system architecture", func(t *testing.T) {
					t.Parallel()

					dockerHost, _ := host.GetTestDockerHost(t, image)
					ctx := log.WithTestLogger(t.Context())
					agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
					require.NoError(t, err)

					cmd := types.Cmd{
						Path: "/usr/bin/dpkg",
						Args: []string{"--print-architecture"},
					}
					arch := strings.TrimSpace(runAndRequireSuccess(t, ctx, agentHost, cmd))

					aptPackages := &APTPackages{}
					packages := []*APTPackage{
						{
							Package:       "libc6",
							Architectures: []string{arch},
						},
					}
					err = aptPackages.Apply(ctx, agentHost, packages)
					require.NoError(t, err)

					cmd = types.Cmd{
						Path: "/usr/bin/dpkg",
						Args: []string{"-l", "libc6:" + arch},
					}
					stdout := runAndRequireSuccess(t, ctx, agentHost, cmd)
					require.True(t, strings.Contains(stdout, "libc6"), "libc6 package not found in dpkg output: %s", stdout)
				})

				// t.Run("package with foreign architecture", func(t *testing.T) {
				// 	t.Parallel()

				// 	dockerHost, _ := host.GetTestDockerHost(t, image)
				// 	ctx := log.WithTestLogger(t.Context())
				// 	agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
				// 	require.NoError(t, err)

				// 	systemArch := strings.TrimSpace(runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
				// 		Path: "/usr/bin/dpkg",
				// 		Args: []string{"--print-architecture"},
				// 	}))

				// 	foreignArch := "amd64"
				// 	if systemArch == "amd64" {
				// 		foreignArch = "i386"
				// 	}
				// 	// FIXME https://wiki.debian.org/Multiarch/HOWTO
				// 	// This should be handled by a new DpkgArchitecture resource instead
				// 	runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
				// 		Path: "/usr/bin/dpkg",
				// 		Args: []string{"--add-architecture", foreignArch},
				// 	})

				// 	archs := []string{systemArch, foreignArch}

				// 	aptPackages := &APTPackages{}
				// 	packages := []*APTPackage{
				// 		{
				// 			Package:       "libc6",
				// 			Architectures: archs,
				// 		},
				// 	}
				// 	err = aptPackages.Apply(ctx, agentHost, packages)
				// 	require.NoError(t, err)

				// 	for _, arch := range archs {
				// 		stdout := runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
				// 			Path: "/usr/bin/dpkg",
				// 			Args: []string{"-l", "libc6:" + arch},
				// 		})
				// 		require.True(t, strings.Contains(stdout, "libc6"), "libc6 package not found in dpkg output: %s", stdout)
				// 	}
				// })

				// t.Run("package hold", func(t *testing.T) {
				// 	t.Parallel()

				// 	dockerHost, _ := host.GetTestDockerHost(t, image)
				// 	ctx := log.WithTestLogger(t.Context())
				// 	agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
				// 	require.NoError(t, err)

				// 	aptPackages := &APTPackages{}
				// 	packages := []*APTPackage{
				// 		{
				// 			Package: "curl",
				// 		},
				// 	}
				// 	err = aptPackages.Apply(ctx, agentHost, packages)
				// 	require.NoError(t, err)

				// 	// Query the installed version
				// 	cmd := types.Cmd{
				// 		Path: "/usr/bin/dpkg-query",
				// 		Args: []string{"-W", "-f", "${Version}", "curl"},
				// 	}
				// 	version := runAndRequireSuccess(t, ctx, agentHost, cmd)
				// 	require.NotEmpty(t, version, "curl version should not be empty")

				// 	// Now apply with hold set
				// 	packages = []*APTPackage{
				// 		{
				// 			Package: "curl",
				// 			Version: strings.TrimSpace(version),
				// 			Hold:    true,
				// 		},
				// 	}
				// 	err = aptPackages.Apply(ctx, agentHost, packages)
				// 	require.NoError(t, err)

				// 	// Check that curl is on hold
				// 	cmd = types.Cmd{
				// 		Path: "/usr/bin/dpkg",
				// 		Args: []string{"--get-selections", "curl"},
				// 	}
				// 	stdout := runAndRequireSuccess(t, ctx, agentHost, cmd)
				// 	require.True(t, strings.Contains(stdout, "hold"), "curl package should be on hold: %s", stdout)
				// })

				// t.Run("package with debconf selections", func(t *testing.T) {
				// 	t.Parallel()

				// 	dockerHost, _ := host.GetTestDockerHost(t, image)
				// 	ctx := log.WithTestLogger(t.Context())
				// 	agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
				// 	require.NoError(t, err)

				// 	aptPackages := &APTPackages{}
				// 	packages := []*APTPackage{
				// 		{
				// 			Package: "tzdata",
				// 			DebconfSelections: map[DebconfQuestion]DebconfAnswer{
				// 				"tzdata/Areas":        "Europe",
				// 				"tzdata/Zones/Europe": "Dublin",
				// 			},
				// 		},
				// 	}
				// 	err = aptPackages.Apply(ctx, agentHost, packages)
				// 	require.NoError(t, err)

				// 	// Verify package is installed
				// 	cmd := types.Cmd{
				// 		Path: "/usr/bin/dpkg",
				// 		Args: []string{"-l", "tzdata"},
				// 	}
				// 	stdout := runAndRequireSuccess(t, ctx, agentHost, cmd)
				// 	require.True(t, strings.Contains(stdout, "tzdata"), "tzdata package not found in dpkg output: %s", stdout)

				// 	// Verify debconf selections (if debconf-show is available)
				// 	cmd = types.Cmd{
				// 		Path: "debconf-show",
				// 		Args: []string{"tzdata"},
				// 	}
				// 	waitStatus, stdout, _, err := lib.SimpleRun(ctx, agentHost, cmd)
				// 	if err == nil && waitStatus.Success() {
				// 		require.True(t, strings.Contains(stdout, "tzdata/Areas"), "debconf selection tzdata/Areas not found: %s", stdout)
				// 	}
				// })

				t.Run("mixed install and remove operations", func(t *testing.T) {
					t.Parallel()

					dockerHost, _ := host.GetTestDockerHost(t, image)
					ctx := log.WithTestLogger(t.Context())
					agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
					require.NoError(t, err)

					aptPackages := &APTPackages{}

					// First install both packages
					packages := []*APTPackage{
						{
							Package: "wget",
						},
						{
							Package: "curl",
						},
					}
					err = aptPackages.Apply(ctx, agentHost, packages)
					require.NoError(t, err)

					// Then mix operations: keep wget, remove curl, install nano
					packages = []*APTPackage{
						{
							Package: "wget", // keep installed
						},
						{
							Package: "curl",
							Absent:  true, // remove
						},
						{
							Package: "nano", // install new
						},
					}
					err = aptPackages.Apply(ctx, agentHost, packages)
					require.NoError(t, err)

					// Verify wget is still installed
					cmd := types.Cmd{
						Path: "/usr/bin/dpkg",
						Args: []string{"-l", "wget"},
					}
					waitStatus, stdout, _, err := lib.SimpleRun(ctx, agentHost, cmd)
					require.NoError(t, err)
					require.True(t, waitStatus.Success() && strings.Contains(stdout, "ii  wget"), "wget should still be installed: %s", stdout)

					// Verify curl is removed
					cmd = types.Cmd{
						Path: "/usr/bin/dpkg",
						Args: []string{"-l", "curl"},
					}
					waitStatus, stdout, _, err = lib.SimpleRun(ctx, agentHost, cmd)
					require.NoError(t, err)
					require.False(t, waitStatus.Success() && strings.Contains(stdout, "ii  curl"), "curl should be removed: %s", stdout)

					// Verify nano is installed
					cmd = types.Cmd{
						Path: "/usr/bin/dpkg",
						Args: []string{"-l", "nano"},
					}
					waitStatus, stdout, _, err = lib.SimpleRun(ctx, agentHost, cmd)
					require.NoError(t, err)
					require.True(t, waitStatus.Success() && strings.Contains(stdout, "ii  nano"), "nano should be installed: %s", stdout)
				})
			})
		}
	})
}
