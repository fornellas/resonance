package host

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
)

func tempDirWithPrefix(t *testing.T, prefix string) string {
	dir, err := os.MkdirTemp(prefix, strings.ReplaceAll(t.Name(), string(os.PathSeparator), "_"))
	require.NoError(t, err)
	return dir
}

func testBaseHost(
	t *testing.T,
	ctx context.Context,
	tempDirPrefix string,
	baseHost types.BaseHost,
	baseHostString,
	baseHostType string,
) {
	t.Run("Run", func(t *testing.T) {
		t.Run("Args, output and failure", func(t *testing.T) {
			waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, baseHost, types.Cmd{
				Path: "ls",
				Args: []string{"-d", "../tmp", "/non-existent"},
			})
			require.NoError(t, err)
			t.Cleanup(func() {
				if t.Failed() {
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
				waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, baseHost, types.Cmd{
					Path: "env",
				})
				t.Cleanup(func() {
					if t.Failed() {
						t.Logf("stdout:\n%s\nstderr:\n%s\n", stdout, stderr)
					}
				})
				require.NoError(t, err)

				require.True(t, waitStatus.Success())
				require.Equal(t, 0, waitStatus.ExitCode)
				require.True(t, waitStatus.Exited)
				require.Equal(t, "", waitStatus.Signal)

				receivedEnv := []string{}
				for _, value := range strings.Split(stdout, "\n") {
					if value == "" {
						continue
					}
					receivedEnv = append(receivedEnv, value)
				}
				sort.Strings(receivedEnv)
				require.Equal(t, types.DefaultEnv, receivedEnv)
				require.Empty(t, stderr)
			})
			t.Run("Set", func(t *testing.T) {
				env := "FOO=bar"
				waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, baseHost, types.Cmd{
					Path: "env",
					Env:  []string{env},
				})
				require.NoError(t, err)
				t.Cleanup(func() {
					if t.Failed() {
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
			t.Run("Success", func(t *testing.T) {
				dir := tempDirWithPrefix(t, tempDirPrefix)
				waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, baseHost, types.Cmd{
					Path: "pwd",
					Dir:  dir,
				})
				require.NoError(t, err)
				t.Cleanup(func() {
					if t.Failed() {
						t.Logf("\nstdout:\n%s\nstderr:\n%s\n", stdout, stderr)
					}
				})
				require.True(t, waitStatus.Success())
				require.Equal(t, 0, waitStatus.ExitCode)
				require.True(t, waitStatus.Exited)
				require.Equal(t, "", waitStatus.Signal)
				require.Equal(t, fmt.Sprintf("%s\n", dir), stdout)
				require.Equal(t, "", stderr)
			})
			t.Run("path must be absolute", func(t *testing.T) {
				_, _, _, err := lib.SimpleRun(ctx, baseHost, types.Cmd{
					Path: "pwd",
					Dir:  "foo/bar",
				})
				require.ErrorContains(t, err, "path must be absolute")
			})
		})
		t.Run("Stdin", func(t *testing.T) {
			stdin := "hello"
			waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, baseHost, types.Cmd{
				Path:  "sh",
				Args:  []string{"-c", "read v && echo =$v="},
				Stdin: strings.NewReader(fmt.Sprintf("%s\n", stdin)),
			})
			require.NoError(t, err)
			t.Cleanup(func() {
				if t.Failed() {
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

	t.Run("String()", func(t *testing.T) {
		require.Equal(t, baseHostString, baseHost.String())
	})

	t.Run("Type()", func(t *testing.T) {
		require.Equal(t, baseHostType, baseHost.Type())
	})

	t.Run("Close()", func(t *testing.T) {
		t.SkipNow()
		require.NoError(t, baseHost.Close(ctx))
	})
}
