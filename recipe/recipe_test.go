package recipe

import "testing"

func TestParseValid(t *testing.T) {
	testParseValid(t, "testdata/recipe/valid")
}

func TestParseInvalid(t *testing.T) {
	testParseInvalid(t, "testdata/recipe/invalid", map[string]string{
		"duplicate_path.yaml":    "testdata/recipe/invalid/duplicate_path.yaml:4:9: file \"/etc/motd\" already declared at testdata/recipe/invalid/duplicate_path.yaml:2:9",
		"duplicate_package.yaml": "testdata/recipe/invalid/duplicate_package.yaml:3:12: apt_package \"nginx\" already declared at testdata/recipe/invalid/duplicate_package.yaml:2:12",
		"malformed_yaml.yaml":    "failed to parse recipe",
	})
}
