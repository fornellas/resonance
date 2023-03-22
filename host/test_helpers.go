package host

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/log"
)

func testHost(t *testing.T, host Host) {
	ctx := context.Background()
	var outputBuffer bytes.Buffer
	ctx = log.SetLoggerValue(ctx, &outputBuffer, "debug", func(code int) {
		t.Fatalf("exit called with %d", code)
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("Args, output and failure", func(t *testing.T) {
			outputBuffer.Reset()
			waitStatus, stdout, stderr, err := host.Run(ctx, Cmd{
				Path: "ls",
				Args: []string{"-d", "../tmp", "/non-existent"},
			})
			require.NoError(t, err)
			t.Cleanup(func() {
				if t.Failed() {
					t.Logf("\noutput:\n%s\n", outputBuffer.String())
					t.Logf("\nstdout:\n%s\nstderr:\n%s\n", stdout, stderr)
				}
			})
			require.False(t, waitStatus.Success())
			require.Equal(t, 2, waitStatus.ExitCode)
			require.True(t, waitStatus.Exited)
			require.Equal(t, "", waitStatus.Signal)
			require.Equal(t, "../tmp\n", stdout)
			require.Equal(t, "ls: cannot access '/non-existent': No such file or directory\n", stderr)
		})
		t.Run("Env", func(t *testing.T) {
			t.Run("Empty", func(t *testing.T) {
				outputBuffer.Reset()
				waitStatus, stdout, stderr, err := host.Run(ctx, Cmd{
					Path: "env",
				})
				require.NoError(t, err)
				require.True(t, waitStatus.Success())
				require.Equal(t, 0, waitStatus.ExitCode)
				require.True(t, waitStatus.Exited)
				require.Equal(t, "", waitStatus.Signal)
				require.Equal(t, "LANG=en_US.UTF-8\n", stdout)
				require.Empty(t, stderr)
			})
			t.Run("Set", func(t *testing.T) {
				outputBuffer.Reset()
				env := "FOO=bar"
				waitStatus, stdout, stderr, err := host.Run(ctx, Cmd{
					Path: "env",
					Env:  []string{env},
				})
				require.NoError(t, err)
				t.Cleanup(func() {
					if t.Failed() {
						t.Logf("output:\n%s\n", outputBuffer.String())
						t.Logf("stdout:\n%s\nstderr:\n%s\n", stdout, stderr)
					}
				})
				require.True(t, waitStatus.Success())
				require.Equal(t, 0, waitStatus.ExitCode)
				require.True(t, waitStatus.Exited)
				require.Equal(t, "", waitStatus.Signal)
				require.Equal(t, env, strings.TrimRight(stdout, "\n"))
				require.Empty(t, stderr)
			})
		})
		t.Run("Dir", func(t *testing.T) {
			outputBuffer.Reset()
			wd, err := os.Getwd()
			require.NoError(t, err)
			waitStatus, stdout, stderr, err := host.Run(ctx, Cmd{
				Path: "pwd",
				Dir:  wd,
			})
			require.NoError(t, err)
			t.Cleanup(func() {
				if t.Failed() {
					t.Logf("\noutput:\n%s\n", outputBuffer.String())
					t.Logf("\nstdout:\n%s\nstderr:\n%s\n", stdout, stderr)
				}
			})
			require.True(t, waitStatus.Success())
			require.Equal(t, 0, waitStatus.ExitCode)
			require.True(t, waitStatus.Exited)
			require.Equal(t, "", waitStatus.Signal)
			require.Equal(t, fmt.Sprintf("%s\n", wd), stdout)
			require.Equal(t, "", stderr)
		})
		t.Run("Stdin", func(t *testing.T) {
			outputBuffer.Reset()
			stdin := "hello"
			waitStatus, stdout, stderr, err := host.Run(ctx, Cmd{
				Path:  "sh",
				Args:  []string{"-c", "read v && echo =$v="},
				Stdin: strings.NewReader(fmt.Sprintf("%s\n", stdin)),
			})
			require.NoError(t, err)
			t.Cleanup(func() {
				if t.Failed() {
					t.Logf("\noutput:\n%s\n", outputBuffer.String())
					t.Logf("\nstdout:\n%s\nstderr:\n%s\n", stdout, stderr)
				}
			})
			require.True(t, waitStatus.Success())
			require.Equal(t, 0, waitStatus.ExitCode)
			require.True(t, waitStatus.Exited)
			require.Equal(t, "", waitStatus.Signal)
			require.Equal(t, fmt.Sprintf("=%s=\n", stdin), stdout)
			require.Equal(t, "", stderr)
		})
	})
}
