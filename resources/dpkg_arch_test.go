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

func TestDpkgArch(t *testing.T) {
	t.Run("Apply()", func(t *testing.T) {
		for _, image := range testDockerImages {
			t.Run(image, func(t *testing.T) {
				t.Parallel()

				dockerHost, _ := host.GetTestDockerHost(t, image)
				ctx := log.WithTestLogger(t.Context())
				host, err := host.NewAgentClientWrapper(ctx, dockerHost)
				require.NoError(t, err)

				// Get system architecture
				systemArch := strings.TrimSpace(runAndRequireSuccess(t, ctx, host, types.Cmd{
					Path: "/usr/bin/dpkg", Args: []string{"--print-architecture"},
				}))

				// Ensure no foreign architectures initially
				initialForeign := strings.TrimSpace(runAndRequireSuccess(t, ctx, host, types.Cmd{
					Path: "/usr/bin/dpkg", Args: []string{"--print-foreign-architectures"},
				}))
				require.Equal(t, "", initialForeign)

				// Set foreign arch
				foreignArch := "i386"
				if systemArch == "i386" {
					foreignArch = "amd64"
				}

				// Add extra foreign architecture
				extraArch := "arm64"
				if systemArch == extraArch {
					extraArch = "armhf"
				}
				runAndRequireSuccess(t, ctx, host, types.Cmd{
					Path: "/usr/bin/dpkg", Args: []string{"--add-architecture", extraArch},
				})

				// Apply only one foreign architecture
				dpkgArch := &DpkgArch{ForeignArchitectures: []string{foreignArch}}
				require.NoError(t, dpkgArch.Apply(ctx, host))

				// Check that the foreign architecture was added
				foreignOut := strings.TrimSpace(runAndRequireSuccess(t, ctx, host, types.Cmd{
					Path: "/usr/bin/dpkg", Args: []string{"--print-foreign-architectures"},
				}))
				require.Equal(t, foreignArch, foreignOut)

				// Install a package for the foreign architecture
				foreignPackage := "libc6:" + foreignArch
				runAndRequireSuccess(t, ctx, host, types.Cmd{
					Path: "/usr/bin/apt", Args: []string{"update"},
				})
				runAndRequireSuccess(t, ctx, host, types.Cmd{
					Path: "/usr/bin/apt", Args: []string{"-y", "install", foreignPackage},
				})

				// Remove all foreign architectures (should purge the package)
				dpkgArch.ForeignArchitectures = []string{}
				require.NoError(t, dpkgArch.Apply(ctx, host))

				// Check that foreign package was removed
				waitStatus, _, stderr, err := lib.Run(ctx, host, types.Cmd{
					Path: "/usr/bin/dpkg", Args: []string{"-l", foreignPackage},
				})
				require.NoError(t, err)
				require.False(t, waitStatus.Success())
				require.Contains(t, stderr, "no packages found")

				// Check that the foreign architecture was removed
				foreignOut = strings.TrimSpace(runAndRequireSuccess(t, ctx, host, types.Cmd{
					Path: "/usr/bin/dpkg", Args: []string{"--print-foreign-architectures"},
				}))
				require.Equal(t, "", foreignOut)

				// Should fail if foreign arch matches system arch
				dpkgArch.ForeignArchitectures = []string{systemArch}
				err = dpkgArch.Apply(ctx, host)
				require.Error(t, err)
				require.Contains(t, err.Error(), "matches system architecture")
			})
		}
	})
	t.Run("Satisfies()", func(t *testing.T) {
		a := &DpkgArch{ForeignArchitectures: []string{"i386", "arm64"}}
		b := &DpkgArch{ForeignArchitectures: []string{"i386"}}
		require.True(t, a.Satisfies(b))
		require.False(t, b.Satisfies(a))
		require.True(t, a.Satisfies(&DpkgArch{ForeignArchitectures: []string{}}))
		require.True(t, (&DpkgArch{ForeignArchitectures: []string{}}).Satisfies(&DpkgArch{ForeignArchitectures: []string{}}))
		require.False(t, (&DpkgArch{ForeignArchitectures: []string{}}).Satisfies(a))
	})
	t.Run("Validate()", func(t *testing.T) {
		require.NoError(t, (&DpkgArch{ForeignArchitectures: []string{"i386", "amd64"}}).Validate())
		require.Error(t, (&DpkgArch{ForeignArchitectures: []string{"!"}}).Validate())
		require.Error(t, (&DpkgArch{ForeignArchitectures: []string{"amd64", "bad!"}}).Validate())
	})
}
