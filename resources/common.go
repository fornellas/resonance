package resources

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

// FormatSourceLocations returns a human-readable string describing source locations
func FormatSourceLocations(locations []hcl.Range) string {
	if len(locations) == 0 {
		return "unknown source location"
	}
	if len(locations) == 1 {
		loc := locations[0]
		return fmt.Sprintf("%s:%d:%d", loc.Filename, loc.Start.Line, loc.Start.Column)
	}
	// Multiple locations - show them all
	var locationStrings []string
	for _, loc := range locations {
		locationStrings = append(locationStrings, fmt.Sprintf("%s:%d:%d", loc.Filename, loc.Start.Line, loc.Start.Column))
	}
	return fmt.Sprintf("merged from: %s", strings.Join(locationStrings, ", "))
}
