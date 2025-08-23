//go:build !skip_integration

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
)

func TestDpkgAlternativeIntegration(t *testing.T) {
	t.Run("Load()", func(t *testing.T) {
		for _, image := range testDockerImages {
			t.Run(image, func(t *testing.T) {
				t.Parallel()

				dockerHost, _ := host.GetTestDockerHost(t, image)
				ctx := log.WithTestLogger(t.Context())
				agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
				require.NoError(t, err)

				// Ensure nano is installed so editor alternatives exist
				runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "apt", Args: []string{"update"},
				})
				runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "apt", Args: []string{"install", "-y", "nano", "ed"},
				})

				// Load the "editor" alternative group
				alt := &DpkgAlternative{Name: "editor"}
				err = alt.Load(ctx, agentHost)
				require.NoError(t, err)

				// Validate
				require.Equal(t, &DpkgAlternative{
					Name:   "editor",
					Link:   "/usr/bin/editor",
					Status: "auto",
					Value:  "/bin/nano",
					Choices: []AlternativeChoice{
						{
							Alternative: "/bin/ed",
							Priority:    -100,
							Slaves: map[string]string{
								"editor.1.gz": "/usr/share/man/man1/ed.1.gz",
							},
						},
						{
							Alternative: "/bin/nano",
							Priority:    40,
							Slaves: map[string]string{
								"editor.1.gz": "/usr/share/man/man1/nano.1.gz",
							},
						},
					},
				}, alt)
			})
		}
	})
}
