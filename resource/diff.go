package resource

import (
	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/yaml.v3"
)

func Diff(a, b interface{}) []diffmatchpatch.Diff {
	var aStr string
	if a != nil {
		aBytes, err := yaml.Marshal(a)
		if err != nil {
			panic(err)
		}
		aStr = string(aBytes)
	}

	var bStr string
	if b != nil {
		bBytes, err := yaml.Marshal(b)
		if err != nil {
			panic(err)
		}
		bStr = string(bBytes)
	}

	return diffmatchpatch.New().DiffMain(aStr, bStr, false)
}

// DiffsHasChanges return true when the diff contains no changes.
func DiffsHasChanges(diffs []diffmatchpatch.Diff) bool {
	for _, diff := range diffs {
		if diff.Type != diffmatchpatch.DiffEqual {
			return true
		}
	}
	return false
}
