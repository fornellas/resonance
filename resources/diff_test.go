package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var a = `hello
world`

var b = `hello
again
world`

var d = `|-
    hello
+    again
    world
`

func TestDiffAsYaml(t *testing.T) {
	chunks := DiffAsYaml(a, b)
	require.Equal(t, d, chunks.String())
}
