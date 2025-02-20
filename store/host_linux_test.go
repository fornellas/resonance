package store

import (
	"testing"

	hostPkg "github.com/fornellas/resonance/host"
)

func TestHostStore(t *testing.T) {
	host := hostPkg.Local{}

	store := NewHostStore(host, t.TempDir())

	testStore(t, store)
}
