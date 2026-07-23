package recipe

import "testing"

func TestFileValid(t *testing.T) {
	testParseValid(t, "testdata/file/valid")
}

func TestFileInvalid(t *testing.T) {
	testParseInvalid(t, "testdata/file/invalid", map[string]string{
		"missing_path.yaml":             "testdata/file/invalid/missing_path.yaml:2:9: path is required",
		"bad_mode_no_leading_zero.yaml": "testdata/file/invalid/bad_mode_no_leading_zero.yaml:2:9: mode \"644\" must be in octal notation",
		"bad_mode_invalid_digit.yaml":   "testdata/file/invalid/bad_mode_invalid_digit.yaml:2:9: mode \"0894\" must be in octal notation",
	})
}
