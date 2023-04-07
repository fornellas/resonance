package resource

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

func TestDiff(t *testing.T) {
	chunks := Diff(a, b)
	require.Equal(t, d, chunks.String())
}
