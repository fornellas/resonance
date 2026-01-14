//go:build !skip_integration

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/lib"
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
					Slaves: map[string]string{
						"editor.1.gz": "/usr/share/man/man1/editor.1.gz",
					},
					Choices: []DpkgAlternativeChoice{
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

				// Test missing alternative
				missing := &DpkgAlternative{Name: "idontexist"}
				err = missing.Load(ctx, agentHost)
				require.NoError(t, err)
				require.Equal(t, &DpkgAlternative{
					Name:   "idontexist",
					Absent: true,
				}, missing)
			})
		}
	})

	t.Run("Apply()", func(t *testing.T) {
		for _, image := range testDockerImages {
			t.Run(image, func(t *testing.T) {
				t.Parallel()

				dockerHost, _ := host.GetTestDockerHost(t, image)
				ctx := log.WithTestLogger(t.Context())
				agentHost, err := host.NewAgentClientWrapper(ctx, dockerHost)
				require.NoError(t, err)

				// Ensure nano and ed are installed
				runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "apt", Args: []string{"update"},
				})
				runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "apt", Args: []string{"install", "-y", "nano", "ed"},
				})

				// Clean up any previous alternatives
				runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "update-alternatives", Args: []string{"--remove-all", "editor"},
				})

				// Add nano as editor with priority 40
				alt := &DpkgAlternative{
					Name: "editor",
					Link: "/usr/bin/editor",
					Slaves: map[string]string{
						"editor.1.gz": "/usr/share/man/man1/editor.1.gz",
					},
					Choices: []DpkgAlternativeChoice{
						{
							Alternative: "/bin/nano",
							Priority:    40,
							Slaves: map[string]string{
								"editor.1.gz": "/usr/share/man/man1/nano.1.gz",
							},
						},
					},
				}
				require.NoError(t, alt.Apply(ctx, agentHost))
				out := runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "update-alternatives", Args: []string{"--query", "editor"},
				})
				expected := `Name: editor
Link: /usr/bin/editor
Slaves:
 editor.1.gz /usr/share/man/man1/editor.1.gz
Status: auto
Best: /bin/nano
Value: /bin/nano

Alternative: /bin/nano
Priority: 40
Slaves:
 editor.1.gz /usr/share/man/man1/nano.1.gz
`
				require.Equal(t, expected, out)

				// Link
				alt.Link = "/usr/bin/fooeditor"
				require.NoError(t, alt.Apply(ctx, agentHost))
				out = runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "update-alternatives", Args: []string{"--query", "editor"},
				})
				expected = `Name: editor
Link: /usr/bin/fooeditor
Slaves:
 editor.1.gz /usr/share/man/man1/editor.1.gz
Status: auto
Best: /bin/nano
Value: /bin/nano

Alternative: /bin/nano
Priority: 40
Slaves:
 editor.1.gz /usr/share/man/man1/nano.1.gz
`
				require.Equal(t, expected, out)

				// Slaves
				alt.Slaves["fooeditor.1.gz"] = "/usr/share/man/man1/fooeditor.1.gz"
				alt.Choices[0].Slaves["fooeditor.1.gz"] = "/usr/share/man/man1/foonano.1.gz"
				require.NoError(t, alt.Apply(ctx, agentHost))
				out = runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "update-alternatives", Args: []string{"--query", "editor"},
				})
				expected = `Name: editor
Link: /usr/bin/fooeditor
Slaves:
 editor.1.gz /usr/share/man/man1/editor.1.gz
 fooeditor.1.gz /usr/share/man/man1/fooeditor.1.gz
Status: auto
Best: /bin/nano
Value: /bin/nano

Alternative: /bin/nano
Priority: 40
Slaves:
 editor.1.gz /usr/share/man/man1/nano.1.gz
 fooeditor.1.gz /usr/share/man/man1/foonano.1.gz
`
				require.Equal(t, expected, out)

				// Status / Value
				alt.Status = "manual"
				alt.Value = "/bin/nano"
				require.NoError(t, alt.Apply(ctx, agentHost))
				out = runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "update-alternatives", Args: []string{"--query", "editor"},
				})
				expected = `Name: editor
Link: /usr/bin/fooeditor
Slaves:
 editor.1.gz /usr/share/man/man1/editor.1.gz
 fooeditor.1.gz /usr/share/man/man1/fooeditor.1.gz
Status: manual
Best: /bin/nano
Value: /bin/nano

Alternative: /bin/nano
Priority: 40
Slaves:
 editor.1.gz /usr/share/man/man1/nano.1.gz
 fooeditor.1.gz /usr/share/man/man1/foonano.1.gz
`
				require.Equal(t, expected, out)

				// Choices
				alt.Choices = append(alt.Choices, DpkgAlternativeChoice{
					Alternative: "/bin/ed",
					Priority:    -100,
					Slaves: map[string]string{
						"editor.1.gz": "/usr/share/man/man1/ed.1.gz",
					},
				})
				require.NoError(t, alt.Apply(ctx, agentHost))
				out = runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "update-alternatives", Args: []string{"--query", "editor"},
				})
				expected = `Name: editor
Link: /usr/bin/fooeditor
Slaves:
 editor.1.gz /usr/share/man/man1/editor.1.gz
 fooeditor.1.gz /usr/share/man/man1/fooeditor.1.gz
Status: manual
Best: /bin/nano
Value: /bin/nano

Alternative: /bin/ed
Priority: -100
Slaves:
 editor.1.gz /usr/share/man/man1/ed.1.gz

Alternative: /bin/nano
Priority: 40
Slaves:
 editor.1.gz /usr/share/man/man1/nano.1.gz
 fooeditor.1.gz /usr/share/man/man1/foonano.1.gz
`
				require.Equal(t, expected, out)

				// Choice
				alt.Choices = []DpkgAlternativeChoice{
					{
						Alternative: "/bin/ed",
						Priority:    -200,
						Slaves: map[string]string{
							"editor.1.gz":    "/usr/share/man/man1/ed.1.gz",
							"fooeditor.1.gz": "/usr/share/man/man1/foobared.1.gz",
						},
					},
					{
						Alternative: "/bin/nano",
						Priority:    80,
						Slaves: map[string]string{
							"editor.1.gz":    "/usr/share/man/man1/nano.1.gz",
							"fooeditor.1.gz": "/usr/share/man/man1/foobarnano.1.gz",
						},
					},
				}
				require.NoError(t, alt.Apply(ctx, agentHost))
				out = runAndRequireSuccess(t, ctx, agentHost, types.Cmd{
					Path: "update-alternatives", Args: []string{"--query", "editor"},
				})
				expected = `Name: editor
Link: /usr/bin/fooeditor
Slaves:
 editor.1.gz /usr/share/man/man1/editor.1.gz
 fooeditor.1.gz /usr/share/man/man1/fooeditor.1.gz
Status: manual
Best: /bin/nano
Value: /bin/nano

Alternative: /bin/ed
Priority: -200
Slaves:
 editor.1.gz /usr/share/man/man1/ed.1.gz
 fooeditor.1.gz /usr/share/man/man1/foobared.1.gz

Alternative: /bin/nano
Priority: 80
Slaves:
 editor.1.gz /usr/share/man/man1/nano.1.gz
 fooeditor.1.gz /usr/share/man/man1/foobarnano.1.gz
`
				require.Equal(t, expected, out)

				// Absent
				alt = &DpkgAlternative{
					Name:   "editor",
					Absent: true,
				}
				require.NoError(t, alt.Apply(ctx, agentHost))
				waitStatus, stdout, stderr, err := lib.Run(ctx, dockerHost, types.Cmd{
					Path: "update-alternatives", Args: []string{"--query", "editor"},
				})
				require.NoError(t, err)
				require.Equal(t, uint32(2), waitStatus.ExitCode)
				require.Equal(t, stdout, "")
				require.Equal(t, stderr, "update-alternatives: error: no alternatives for editor\n")
			})
		}
	})
}
