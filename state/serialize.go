package state

import (
	"io"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// SerializeToHCL converts a State object to HCL format and returns the bytes.
//
// This function serializes all resources in the state back to HCL configuration format.
// The serialization process automatically excludes metadata fields such as SourceLocation
// (which has the `def_range` HCL tag) since these are only used for tracking the original
// source location during parsing and are not part of the actual configuration data.
//
// The output includes:
//   - All File resources as "file" blocks
//   - All APTPackage resources as "apt_package" blocks
//   - All configuration attributes with their current values
//   - Default values for fields that weren't explicitly set in the original configuration
//
// Note: The serialized HCL may include more fields than the original configuration
// if those fields have default values in the Go structs.
//
// Example usage:
//
//	state, _ := Load([]string{"config.hcl"})
//	hclBytes := SerializeToHCL(state)
//	fmt.Print(string(hclBytes))
func SerializeToHCL(state *State) []byte {
	hclFile := hclwrite.NewEmptyFile()
	body := hclFile.Body()

	// Encode all File resources
	for _, file := range state.Files {
		fileBlock := gohcl.EncodeAsBlock(file, "file")
		body.AppendBlock(fileBlock)
	}

	// Encode all APTPackage resources
	for _, aptPackage := range state.APTPackages {
		aptBlock := gohcl.EncodeAsBlock(aptPackage, "apt_package")
		body.AppendBlock(aptBlock)
	}

	return hclFile.Bytes()
}

// WriteToHCL converts a State object to HCL format and writes it to the provided writer.
//
// This is a convenience function that combines SerializeToHCL with writing to an io.Writer.
// It uses the same serialization logic as SerializeToHCL, automatically excluding metadata
// fields like SourceLocation that are only used for source tracking.
//
// Parameters:
//   - state: The State object to serialize
//   - writer: The io.Writer to write the HCL content to (e.g., file, buffer, stdout)
//
// Returns:
//   - int: The number of bytes written
//   - error: Any error encountered during the write operation
//
// Example usage:
//
//	state, _ := Load([]string{"config.hcl"})
//	file, _ := os.Create("output.hcl")
//	defer file.Close()
//	bytesWritten, err := WriteToHCL(state, file)
func WriteToHCL(state *State, writer io.Writer) (int, error) {
	hclBytes := SerializeToHCL(state)
	return writer.Write(hclBytes)
}
