package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDpkgAlternative(t *testing.T) {
	t.Run("Validate()", func(t *testing.T) {
		valid := DpkgAlternative{
			Name:   "editor",
			Link:   "/usr/bin/editor",
			Slaves: map[string]string{"editor.1.gz": "/usr/share/man/man1/editor.1.gz"},
			Status: "auto",
			Choices: []DpkgAlternativeChoice{
				{
					Alternative: "/usr/bin/vim.basic",
					Priority:    50,
					Slaves:      map[string]string{"editor.1.gz": "/usr/share/man/man1/vim.1.gz"},
				},
			},
		}
		require := require.New(t)

		// Valid auto mode (Value must be empty)
		require.NoError(valid.Validate())

		// Valid manual mode (Value must be set and valid)
		manual := valid
		manual.Status = "manual"
		manual.Value = "/usr/bin/vim.basic"
		require.NoError(manual.Validate())

		// Manual mode: Value must be set
		manualWithoutValue := valid
		manualWithoutValue.Status = "manual"
		manualWithoutValue.Value = ""
		require.ErrorContains(manualWithoutValue.Validate(), "Value must be set when Status is manual")

		// Name must not be empty
		invalid := valid
		invalid.Name = ""
		require.ErrorContains(invalid.Validate(), "Name is empty")

		// Link must be absolute
		invalid = valid
		invalid.Link = "usr/bin/editor"
		require.ErrorContains(invalid.Validate(), "Link is not absolute")

		// Link must be clean
		invalid = valid
		invalid.Link = "/usr/bin/../bin/editor"
		require.ErrorContains(invalid.Validate(), "Link is not clean")

		// Status must be valid
		invalid = valid
		invalid.Status = "badstatus"
		require.ErrorContains(invalid.Validate(), "invalid Status")

		invalid = valid
		invalid.Status = ""
		require.ErrorContains(invalid.Validate(), "invalid Status")

		// Auto mode: Value must be empty
		invalid = valid
		invalid.Value = "/usr/bin/../bin/vim.basic"
		require.ErrorContains(invalid.Validate(), "Value must be empty when Status is auto")

		invalid = valid
		invalid.Status = "auto"
		invalid.Value = "/usr/bin/vim.basic"
		require.ErrorContains(invalid.Validate(), "Value must be empty when Status is auto")

		invalid = valid
		invalid.Status = "auto"
		invalid.Value = "none"
		require.ErrorContains(invalid.Validate(), "Value must be empty when Status is auto")

		// Manual mode: Value must be absolute
		invalid = valid
		invalid.Status = "manual"
		invalid.Value = "usr/bin/vim.basic"
		require.ErrorContains(invalid.Validate(), "Value is not absolute")

		// Slaves validation
		invalid = valid
		invalid.Slaves = map[string]string{"": "/usr/share/man/man1/editor.1.gz"}
		require.ErrorContains(invalid.Validate(), "slave name is empty")

		invalid = valid
		invalid.Slaves = map[string]string{"editor.1.gz": ""}
		require.ErrorContains(invalid.Validate(), "slave path for editor.1.gz is empty")

		invalid = valid
		invalid.Slaves = map[string]string{"editor.1.gz": "share/man/man1/editor.1.gz"}
		require.ErrorContains(invalid.Validate(), "slave path for editor.1.gz is not absolute")

		invalid = valid
		invalid.Slaves = map[string]string{"editor.1.gz": "/usr/share/man/../man1/editor.1.gz"}
		require.ErrorContains(invalid.Validate(), "slave path for editor.1.gz is not clean")

		// Slave must exist in at least one Choices
		invalid = valid
		invalid.Slaves = map[string]string{"fooeditor.1.gz": "/usr/share/man/man1/fooeditor.1.gz"}
		require.ErrorContains(invalid.Validate(), `slave "fooeditor.1.gz" in Slaves is missing from all Choices`)

		// Valid: slave exists in Choices
		validWithSlave := valid
		validWithSlave.Slaves = map[string]string{
			"editor.1.gz":    "/usr/share/man/man1/editor.1.gz",
			"fooeditor.1.gz": "/usr/share/man/man1/fooeditor.1.gz",
		}
		validWithSlave.Choices = []DpkgAlternativeChoice{
			{
				Alternative: "/usr/bin/vim.basic",
				Priority:    50,
				Slaves: map[string]string{
					"editor.1.gz":    "/usr/share/man/man1/vim.1.gz",
					"fooeditor.1.gz": "/usr/share/man/man1/fooeditor.1.gz",
				},
			},
		}
		require.NoError(validWithSlave.Validate())

		t.Run("AlternativeChoice", func(t *testing.T) {
			valid := DpkgAlternativeChoice{
				Alternative: "/usr/bin/vim.basic",
				Priority:    50,
				Slaves:      map[string]string{"editor.1.gz": "/usr/share/man/man1/vim.1.gz"},
			}
			require.NoError(valid.Validate())

			invalid := valid
			invalid.Alternative = ""
			require.ErrorContains(invalid.Validate(), "alternative path is empty")

			invalid = valid
			invalid.Alternative = "usr/bin/vim.basic"
			require.ErrorContains(invalid.Validate(), "alternative path is not absolute")

			invalid = valid
			invalid.Alternative = "/usr/bin/../bin/vim.basic"
			require.ErrorContains(invalid.Validate(), "alternative path is not clean")

			invalid = valid
			invalid.Slaves = map[string]string{"": "/usr/share/man/man1/vim.1.gz"}
			require.ErrorContains(invalid.Validate(), "slave name is empty")

			invalid = valid
			invalid.Slaves = map[string]string{"editor.1.gz": ""}
			require.ErrorContains(invalid.Validate(), "slave path for editor.1.gz is empty")

			invalid = valid
			invalid.Slaves = map[string]string{"editor.1.gz": "share/man/man1/vim.1.gz"}
			require.ErrorContains(invalid.Validate(), "slave path for editor.1.gz is not absolute")

			invalid = valid
			invalid.Slaves = map[string]string{"editor.1.gz": "/usr/share/man/../man1/vim.1.gz"}
			require.ErrorContains(invalid.Validate(), "slave path for editor.1.gz is not clean")
		})
	})

}
