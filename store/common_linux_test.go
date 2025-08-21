package store

import (
	"context"
	"testing"

	"github.com/fornellas/slogxt/log"
)

func testStore(t *testing.T, store Store) {
	ctx := context.Background()
	_ = log.WithTestLogger(ctx)

	t.SkipNow()
}
