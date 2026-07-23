package recipe

import "testing"

// APTPackage has no YAML entry point of its own, so these fixtures each
// wrap a single package in a minimal recipe, just to reach
// APTPackage.Validate via Parse.
func TestAPTPackageValid(t *testing.T) {
	testParseValid(t, "testdata/apt_package/valid")
}

func TestAPTPackageInvalid(t *testing.T) {
	testParseInvalid(t, "testdata/apt_package/invalid", map[string]string{
		"missing_package.yaml":            "testdata/apt_package/invalid/missing_package.yaml:2:12: package is required",
		"bad_package_name_uppercase.yaml": `testdata/apt_package/invalid/bad_package_name_uppercase.yaml:2:12: package "Nginx" is not a valid package name`,
		"bad_package_name_too_short.yaml": `testdata/apt_package/invalid/bad_package_name_too_short.yaml:2:12: package "n" is not a valid package name`,
	})
}
