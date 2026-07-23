package recipe

import (
	"context"
	"fmt"
	"strconv"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

// File declares the desired state of a single file on the host.
type File struct {
	Path    string `yaml:"path" json:"path" jsonschema:"required,description=Absolute path to the file."`
	Content string `yaml:"content" json:"content,omitempty" jsonschema:"description=The file's desired content. Defaults to an empty file."`
	Mode    string `yaml:"mode" json:"mode" jsonschema:"required,pattern=^0[0-7]{3}[0-7]?$,description=File permissions in octal notation (eg: 0644)."`

	// Location is where this File was declared.
	Location Location `yaml:"-" json:"-"`
}

// UnmarshalYAML implements yaml.NodeUnmarshalerContext: it decodes File as
// usual, additionally capturing where it was declared, in a single pass —
// no separate lookup against the source is needed afterwards.
func (f *File) UnmarshalYAML(ctx context.Context, node ast.Node) error {
	type PlainFile File
	var plainFile PlainFile
	if err := yaml.NodeToValue(node, &plainFile); err != nil {
		return err
	}
	*f = File(plainFile)
	f.Location = locationFromNode(ctx, node)

	return nil
}

// Validate checks that the File is well formed.
func (f File) Validate() error {
	if f.Path == "" {
		return fmt.Errorf("path is required")
	}
	if !isValidMode(f.Mode) {
		return fmt.Errorf("mode %q must be in octal notation (eg: 0644)", f.Mode)
	}
	return nil
}

// isValidMode reports whether mode is a valid file mode in octal notation,
// with or without the setuid/setgid/sticky bit (eg: 0644, 04755): a leading
// "0" followed by 3 or 4 octal digits. The length check bounds the parsed
// value to the valid file mode range (0 to 07777) on its own, so no separate
// range check is needed after parsing.
func isValidMode(mode string) bool {
	if len(mode) < 4 || len(mode) > 5 || mode[0] != '0' {
		return false
	}
	_, err := strconv.ParseUint(mode[1:], 8, 32)
	return err == nil
}
