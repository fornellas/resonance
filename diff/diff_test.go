package diff

import (
	"testing"

	"github.com/kylelemons/godebug/diff"
	"github.com/stretchr/testify/require"
)

func TestChunks(t *testing.T) {
	t.Run("HasChanges", func(t *testing.T) {
		t.Run("empty_chunks", func(t *testing.T) {
			c := Chunks{}
			require.False(t, c.HasChanges())
		})

		t.Run("no_changes", func(t *testing.T) {
			c := Chunks{
				diff.Chunk{Equal: []string{"line1", "line2"}},
				diff.Chunk{Equal: []string{"line3"}},
			}
			require.False(t, c.HasChanges())
		})

		t.Run("with_additions", func(t *testing.T) {
			c := Chunks{
				diff.Chunk{Added: []string{"new line"}},
			}
			require.True(t, c.HasChanges())
		})

		t.Run("with_deletions", func(t *testing.T) {
			c := Chunks{
				diff.Chunk{Deleted: []string{"removed line"}},
			}
			require.True(t, c.HasChanges())
		})

		t.Run("with_mixed_changes", func(t *testing.T) {
			c := Chunks{
				diff.Chunk{Equal: []string{"line1"}},
				diff.Chunk{Added: []string{"new line"}},
				diff.Chunk{Deleted: []string{"old line"}},
			}
			require.True(t, c.HasChanges())
		})
	})

	t.Run("TerminalString", func(t *testing.T) {
		testCases := []struct {
			name     string
			chunks   Chunks
			expected string
		}{
			{
				name:     "empty_chunks",
				chunks:   Chunks{},
				expected: "",
			},
			{
				name: "only_equal",
				chunks: Chunks{
					diff.Chunk{Equal: []string{"line1", "line2"}},
				},
				expected: "line1\nline2\n",
			},
			{
				name: "only_additions",
				chunks: Chunks{
					diff.Chunk{Added: []string{"new line1", "new line2"}},
				},
				expected: "\x1b[32m+new line1\x1b[0m\n\x1b[32m+new line2\x1b[0m\n",
			},
			{
				name: "only_deletions",
				chunks: Chunks{
					diff.Chunk{Deleted: []string{"old line1", "old line2"}},
				},
				expected: "-old line1\n-old line2\n",
			},
			{
				name: "mixed_changes",
				chunks: Chunks{
					diff.Chunk{
						Added:   []string{"new line"},
						Deleted: []string{"old line"},
						Equal:   []string{"unchanged"},
					},
				},
				expected: "\x1b[32m+new line\x1b[0m\n-old line\nunchanged\n",
			},
			{
				name: "empty_lines_at_boundaries",
				chunks: Chunks{
					diff.Chunk{Added: []string{"", "content", ""}},
				},
				expected: "\x1b[32m+content\x1b[0m\n",
			},
			{
				name: "empty_lines_in_middle_chunks",
				chunks: Chunks{
					diff.Chunk{Equal: []string{"before"}},
					diff.Chunk{Added: []string{"", "content", ""}},
					diff.Chunk{Equal: []string{"after"}},
				},
				expected: "before\n\x1b[32m+\x1b[0m\n\x1b[32m+content\x1b[0m\n\x1b[32m+\x1b[0m\nafter\n",
			},
			{
				name: "multiple_chunks",
				chunks: Chunks{
					diff.Chunk{Equal: []string{"line1"}},
					diff.Chunk{Added: []string{"added"}},
					diff.Chunk{Deleted: []string{"removed"}},
					diff.Chunk{Equal: []string{"line2"}},
				},
				expected: "line1\n\x1b[32m+added\x1b[0m\n-removed\nline2\n",
			},
			{
				name: "empty_strings_in_chunks",
				chunks: Chunks{
					diff.Chunk{Added: []string{""}},
					diff.Chunk{Deleted: []string{""}},
					diff.Chunk{Equal: []string{""}},
				},
				expected: "-\n",
			},
			{
				name: "boundary_empty_lines_skipped",
				chunks: Chunks{
					diff.Chunk{Added: []string{"", "content1", ""}},
					diff.Chunk{Equal: []string{"middle"}},
					diff.Chunk{Deleted: []string{"", "content2", ""}},
				},
				expected: "\x1b[32m+content1\x1b[0m\nmiddle\n-content2\n",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.chunks.TerminalString()
				if tc.expected == "" {
					require.Empty(t, result)
				} else {
					require.Equal(t, tc.expected, result)
				}
			})
		}
	})
}
