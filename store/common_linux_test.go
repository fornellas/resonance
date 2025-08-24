package store

import (
	"testing"

	"github.com/fornellas/slogxt/log"
)

func testStore(t *testing.T, store Store) {
	ctx := t.Context()
	_ = log.WithTestLogger(ctx)

	t.SkipNow()
}
