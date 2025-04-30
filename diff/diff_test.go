package diff

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
	diff := DiffAsYaml(a, b)
	require.Equal(t, d, diff.String())
}
