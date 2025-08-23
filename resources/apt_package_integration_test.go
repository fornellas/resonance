//go:build !skip_integration

package resources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
)

func TestAPTPackagesIntegration(t *testing.T) {
	t.Run("Load()", func(t *testing.T) {
		for _, image := range testDockerImages {
			t.Run(image, func(t *testing.T) {
				t.Parallel()
				var err error
				var systemArch string
				dockerHost, _ := host.GetTestDockerHost(t, image)
				ctx := log.WithTestLogger(t.Context())
				agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
				require.NoError(t, err)

				// Fetch system arch
				systemArch = strings.TrimSpace(runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "/usr/bin/dpkg", Args: []string{"--print-architecture"},
				}))

				// APT update
				runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "apt", Args: []string{"update"},
				})

				// Package + Architectures
				runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "apt", Args: []string{"install", "-y", "nano"},
				})
				aptPackages := &APTPackages{}
				nanoPkg := &APTPackage{Package: "nano"}
				err = aptPackages.Load(ctx, agentHost, []*APTPackage{nanoPkg})
				require.NoError(t, err)
				expectedNano := &APTPackage{
					Package:       "nano",
					Architectures: []string{systemArch},
				}
				require.Equal(t, expectedNano, nanoPkg)

				// Absent
				curlAbsent := &APTPackage{Package: "curl", Absent: true}
				err = aptPackages.Load(ctx, agentHost, []*APTPackage{curlAbsent})
				require.NoError(t, err)
				expectedCurlAbsent := &APTPackage{
					Package: "curl",
					Absent:  true,
				}
				require.Equal(t, expectedCurlAbsent, curlAbsent)

				// Version + Hold
				runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "apt-get", Args: []string{"install", "-y", "curl"},
				})
				runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "/usr/bin/dpkg", Args: []string{"--set-selections"},
					Stdin: strings.NewReader("curl hold\n"),
				})
				curlHold := &APTPackage{Package: "curl", Hold: true}
				err = aptPackages.Load(ctx, agentHost, []*APTPackage{curlHold})
				require.NoError(t, err)
				version := strings.TrimSpace(runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "/usr/bin/dpkg-query", Args: []string{"-W", "-f", "${Version}", "curl"},
				}))
				expectedCurlHold := &APTPackage{
					Package:       "curl",
					Architectures: []string{systemArch},
					Version:       version,
					Hold:          true,
				}
				require.Equal(t, expectedCurlHold, curlHold)

				// DebconfSelections
				runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "apt", Args: []string{"install", "-y", "tzdata"},
				})
				runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path:  "debconf-set-selections",
					Stdin: strings.NewReader("tzdata tzdata/Areas select Europe\ntzdata tzdata/Zones/Europe select Dublin\n"),
				})
				tzdataPkg := &APTPackage{
					Package: "tzdata",
				}
				err = aptPackages.Load(ctx, agentHost, []*APTPackage{tzdataPkg})
				require.NoError(t, err)
				require.Contains(t, tzdataPkg.DebconfSelections, DebconfQuestion("tzdata/Areas"))
				require.Equal(t, DebconfAnswer("Europe"), tzdataPkg.DebconfSelections["tzdata/Areas"])
				require.Contains(t, tzdataPkg.DebconfSelections, DebconfQuestion("tzdata/Zones/Europe"))
				require.Equal(t, DebconfAnswer("Dublin"), tzdataPkg.DebconfSelections["tzdata/Zones/Europe"])
				for k := range tzdataPkg.DebconfSelections {
					require.True(t, strings.HasPrefix(string(k), "tzdata/"), "unexpected debconf key: %s", k)
				}
			})
		}
	})
	t.Run("Apply()", func(t *testing.T) {
		for _, image := range testDockerImages {
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
					waitStatus, stdout, _, err := lib.Run(ctx, agentHost, cmd)
					require.NoError(t, err)
					require.False(t, waitStatus.Success() && strings.Contains(stdout, "ii  nano"), "nano package should be removed but found in dpkg output: %s", stdout)
				})

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
					waitStatus, stdout, _, err := lib.Run(ctx, agentHost, cmd)
					require.NoError(t, err)
					require.True(t, waitStatus.Success() && strings.Contains(stdout, "ii  wget"), "wget should still be installed: %s", stdout)

					// Verify curl is removed
					cmd = types.Cmd{
						Path: "/usr/bin/dpkg",
						Args: []string{"-l", "curl"},
					}
					waitStatus, stdout, _, err = lib.Run(ctx, agentHost, cmd)
					require.NoError(t, err)
					require.False(t, waitStatus.Success() && strings.Contains(stdout, "ii  curl"), "curl should be removed: %s", stdout)

					// Verify nano is installed
					cmd = types.Cmd{
						Path: "/usr/bin/dpkg",
						Args: []string{"-l", "nano"},
					}
					waitStatus, stdout, _, err = lib.Run(ctx, agentHost, cmd)
					require.NoError(t, err)
					require.True(t, waitStatus.Success() && strings.Contains(stdout, "ii  nano"), "nano should be installed: %s", stdout)
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

				t.Run("package with foreign architecture", func(t *testing.T) {
					t.Parallel()

					dockerHost, _ := host.GetTestDockerHost(t, image)
					ctx := log.WithTestLogger(t.Context())
					agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
					require.NoError(t, err)

					systemArch := strings.TrimSpace(runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
						Path: "/usr/bin/dpkg",
						Args: []string{"--print-architecture"},
					}))

					foreignArch := "amd64"
					if systemArch == "amd64" {
						foreignArch = "i386"
					}
					runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
						Path: "/usr/bin/dpkg",
						Args: []string{"--add-architecture", foreignArch},
					})

					archs := []string{systemArch, foreignArch}

					aptPackages := &APTPackages{}
					packages := []*APTPackage{
						{
							Package:       "libc6",
							Architectures: archs,
						},
					}
					err = aptPackages.Apply(ctx, agentHost, packages)
					require.NoError(t, err)

					for _, arch := range archs {
						stdout := runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
							Path: "/usr/bin/dpkg",
							Args: []string{"-l", "libc6:" + arch},
						})
						require.True(t, strings.Contains(stdout, "libc6"), "libc6 package not found in dpkg output: %s", stdout)
					}
				})

				t.Run("package version and hold", func(t *testing.T) {
					t.Parallel()

					dockerHost, _ := host.GetTestDockerHost(t, image)
					ctx := log.WithTestLogger(t.Context())
					agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
					require.NoError(t, err)

					aptPackages := &APTPackages{}

					packages := []*APTPackage{
						{
							Package: "curl",
						},
					}
					err = aptPackages.Apply(ctx, agentHost, packages)
					require.NoError(t, err)

					// Query the installed version
					cmd := types.Cmd{
						Path: "/usr/bin/dpkg-query",
						Args: []string{"-W", "-f", "${Version}", "curl"},
					}
					version := runAndRequireSuccess(t, ctx, agentHost, cmd)
					require.NotEmpty(t, version, "curl version should not be empty")

					// Now apply with hold set
					packages = []*APTPackage{
						{
							Package: "curl",
							Version: strings.TrimSpace(version),
							Hold:    true,
						},
					}
					err = aptPackages.Apply(ctx, agentHost, packages)
					require.NoError(t, err)

					// Check that curl is on hold
					cmd = types.Cmd{
						Path: "/usr/bin/dpkg",
						Args: []string{"--get-selections", "curl"},
					}
					stdout := runAndRequireSuccess(t, ctx, agentHost, cmd)
					require.True(t, strings.Contains(stdout, "hold"), "curl package should be on hold: %s", stdout)

					// Remove version & hold
					packages = []*APTPackage{
						{
							Package: "curl",
						},
					}
					err = aptPackages.Apply(ctx, agentHost, packages)
					require.NoError(t, err)

					// Check that curl is not held
					cmd = types.Cmd{
						Path: "/usr/bin/dpkg",
						Args: []string{"--get-selections", "curl"},
					}
					stdout = runAndRequireSuccess(t, ctx, agentHost, cmd)
					require.NotContains(t, stdout, "hold", "curl package should not be held: %s", stdout)
				})

				t.Run("package with debconf selections", func(t *testing.T) {
					t.Parallel()

					dockerHost, _ := host.GetTestDockerHost(t, image)
					ctx := log.WithTestLogger(t.Context())
					agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
					require.NoError(t, err)

					aptPackages := &APTPackages{}

					packages := []*APTPackage{
						{
							Package: "tzdata",
							DebconfSelections: map[DebconfQuestion]DebconfAnswer{
								"tzdata/Areas":        "Europe",
								"tzdata/Zones/Europe": "Dublin",
							},
						},
					}
					err = aptPackages.Apply(ctx, agentHost, packages)
					require.NoError(t, err)

					// Verify debconf selections
					stdout := runAndRequireSuccess(t, ctx, dockerHost, types.Cmd{
						Path: "debconf-show", Args: []string{"tzdata"},
					})
					require.Contains(t, stdout, "* tzdata/Areas: Europe\n")
					require.Contains(t, stdout, "* tzdata/Zones/Europe: Dublin\n")

					// Reconfigure package
					runAndRequireSuccess(t, ctx, dockerHost, types.Cmd{
						Path: "dpkg-reconfigure", Args: []string{"-f", "noninteractive", "tzdata"},
					})

					// Verify debconf selections (reconfiguring must not alter it)
					stdout = runAndRequireSuccess(t, ctx, dockerHost, types.Cmd{
						Path: "debconf-show", Args: []string{"tzdata"},
					})
					require.Contains(t, stdout, "* tzdata/Areas: Europe\n")
					require.Contains(t, stdout, "* tzdata/Zones/Europe: Dublin\n")

					// Change answer for already istalled package
					packages = []*APTPackage{
						{
							Package: "tzdata",
							DebconfSelections: map[DebconfQuestion]DebconfAnswer{
								"tzdata/Areas":        "Europe",
								"tzdata/Zones/Europe": "London",
							},
						},
					}
					err = aptPackages.Apply(ctx, agentHost, packages)
					require.NoError(t, err)

					// Verify debconf selections
					stdout = runAndRequireSuccess(t, ctx, dockerHost, types.Cmd{
						Path: "debconf-show", Args: []string{"tzdata"},
					})
					require.Contains(t, stdout, "* tzdata/Areas: Europe\n")
					require.Contains(t, stdout, "* tzdata/Zones/Europe: London\n")
				})
			})
		}
	})
}
