package resources

import (
	"testing"
)

func TestAPTPackageSatisfies(t *testing.T) {
	tests := []struct {
		name     string
		current  *APTPackage
		target   *APTPackage
		expected bool
	}{
		// Package
		{
			name: "same package, no extra fields",
			current: &APTPackage{
				Package: "wget",
			},
			target: &APTPackage{
				Package: "wget",
			},
			expected: true,
		},
		{
			name: "different package names",
			current: &APTPackage{
				Package: "wget",
			},
			target: &APTPackage{
				Package: "curl",
			},
			expected: false,
		},
		{
			name: "current has architectures, target doesn't - should satisfy",
			current: &APTPackage{
				Package:       "wget",
				Architectures: []string{"amd64"},
			},
			target: &APTPackage{
				Package: "wget",
			},
			expected: true,
		},
		// Absent
		{
			name: "absent mismatch",
			current: &APTPackage{
				Package: "wget",
				Absent:  false,
			},
			target: &APTPackage{
				Package: "wget",
				Absent:  true,
			},
			expected: false,
		},
		{
			name: "absent match",
			current: &APTPackage{
				Package: "wget",
				Absent:  true,
			},
			target: &APTPackage{
				Package: "wget",
				Absent:  true,
			},
			expected: true,
		},
		// Architectures
		{
			name: "current has all required architectures",
			current: &APTPackage{
				Package:       "wget",
				Architectures: []string{"amd64", "i386"},
			},
			target: &APTPackage{
				Package:       "wget",
				Architectures: []string{"amd64"},
			},
			expected: true,
		},
		{
			name: "current has architectures, target has other arch - should not satisfy",
			current: &APTPackage{
				Package:       "wget",
				Architectures: []string{"amd64"},
			},
			target: &APTPackage{
				Package:       "wget",
				Architectures: []string{"i386"},
			},
			expected: false,
		},
		{
			name: "target has architectures, current doesn't - should not satisfy",
			current: &APTPackage{
				Package: "wget",
			},
			target: &APTPackage{
				Package:       "wget",
				Architectures: []string{"amd64"},
			},
			expected: false,
		},

		{
			name: "current missing required architecture",
			current: &APTPackage{
				Package:       "wget",
				Architectures: []string{"i386"},
			},
			target: &APTPackage{
				Package:       "wget",
				Architectures: []string{"amd64"},
			},
			expected: false,
		},
		// Version
		{
			name: "current has version, target doesn't - should satisfy",
			current: &APTPackage{
				Package: "wget",
				Version: "1.21.4-1ubuntu4.1",
			},
			target: &APTPackage{
				Package: "wget",
			},
			expected: true,
		},
		{
			name: "target has version, current doesn't - should not satisfy",
			current: &APTPackage{
				Package: "wget",
			},
			target: &APTPackage{
				Package: "wget",
				Version: "1.21.4-1ubuntu4.1",
			},
			expected: false,
		},
		{
			name: "matching versions",
			current: &APTPackage{
				Package: "wget",
				Version: "1.21.4-1ubuntu4.1",
			},
			target: &APTPackage{
				Package: "wget",
				Version: "1.21.4-1ubuntu4.1",
			},
			expected: true,
		},
		{
			name: "different versions",
			current: &APTPackage{
				Package: "wget",
				Version: "1.21.4-1ubuntu4.1",
			},
			target: &APTPackage{
				Package: "wget",
				Version: "1.21.3-1ubuntu4.1",
			},
			expected: false,
		},
		// DebconfSelections
		{
			name: "target has debconf selections, current doesn't",
			current: &APTPackage{
				Package: "wget",
			},
			target: &APTPackage{
				Package: "wget",
				DebconfSelections: map[DebconfQuestion]DebconfSelection{
					"wget/question": {Answer: "yes", Seen: true},
				},
			},
			expected: false,
		},
		{
			name: "current has debconf selections, target doesn't",
			current: &APTPackage{
				Package: "wget",
				DebconfSelections: map[DebconfQuestion]DebconfSelection{
					"wget/question": {Answer: "yes", Seen: true},
				},
			},
			target: &APTPackage{
				Package: "wget",
			},
			expected: true,
		},
		{
			name: "matching debconf selections",
			current: &APTPackage{
				Package: "wget",
				DebconfSelections: map[DebconfQuestion]DebconfSelection{
					"wget/question": {Answer: "yes", Seen: true},
				},
			},
			target: &APTPackage{
				Package: "wget",
				DebconfSelections: map[DebconfQuestion]DebconfSelection{
					"wget/question": {Answer: "yes", Seen: true},
				},
			},
			expected: true,
		},
		{
			name: "different debconf answers",
			current: &APTPackage{
				Package: "wget",
				DebconfSelections: map[DebconfQuestion]DebconfSelection{
					"wget/question": {Answer: "yes", Seen: true},
				},
			},
			target: &APTPackage{
				Package: "wget",
				DebconfSelections: map[DebconfQuestion]DebconfSelection{
					"wget/question": {Answer: "no", Seen: true},
				},
			},
			expected: false,
		},
		// Misc
		{
			name: "current has both architectures and version, target doesn't - should satisfy",
			current: &APTPackage{
				Package:       "wget",
				Architectures: []string{"amd64"},
				Version:       "1.21.4-1ubuntu4.1",
			},
			target: &APTPackage{
				Package: "wget",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.current.Satisfies(tt.target)
			if result != tt.expected {
				t.Errorf("APTPackage.Satisfies() = %v, want %v", result, tt.expected)
				t.Errorf("Current: %+v", tt.current)
				t.Errorf("Target: %+v", tt.target)
			}
		})
	}
}
