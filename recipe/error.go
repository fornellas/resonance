package recipe

import "fmt"

// ValidationError reports a Recipe validation problem together with its
// location in the source recipe.
type ValidationError struct {
	Location Location
	Err      error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Location, e.Err)
}

func (e *ValidationError) Unwrap() error { return e.Err }

// Location identifies where a resource was declared in a recipe.
type Location struct {
	Path   string
	Line   int
	Column int
}

func (l Location) String() string {
	if l.Line == 0 {
		return l.Path
	}
	return fmt.Sprintf("%s:%d:%d", l.Path, l.Line, l.Column)
}

// DuplicateResourceError reports that the same resource was declared more
// than once — whether twice within a single recipe file, or across several.
type DuplicateResourceError struct {
	// The resource's kind, singular, eg: "file", "apt_package"
	Kind string
	// The resource's identity, eg: a File's Path
	Resource string
	First    Location
	Second   Location
}

func (e *DuplicateResourceError) Error() string {
	return fmt.Sprintf("%s: %s %q already declared at %s", e.Second, e.Kind, e.Resource, e.First)
}

// detectDuplicates errors with a *DuplicateResourceError if any two items
// share the same identity (as computed by key). kind identifies what's
// being checked, singular (eg: "file", "apt_package"), so the error can
// name it unambiguously. declaredAt tracks identities already seen; pass a
// fresh map to check only among items, or a map shared across multiple
// calls to also catch collisions with previously seen items (eg: entries
// from another recipe file already merged in).
func detectDuplicates[T any](
	kind string,
	items []T,
	key func(T) string,
	loc func(T) Location,
	declaredAt map[string]Location,
) error {
	for _, item := range items {
		k := key(item)
		l := loc(item)
		if existing, ok := declaredAt[k]; ok {
			return &DuplicateResourceError{Kind: kind, Resource: k, First: existing, Second: l}
		}
		declaredAt[k] = l
	}
	return nil
}
