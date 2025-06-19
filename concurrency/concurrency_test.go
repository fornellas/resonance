package concurrency

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConcurrency(t *testing.T) {
	t.Run("unlimited", func(t *testing.T) {
		ctx := context.Background()
		cg := NewConcurrencyGroup(ctx)

		var counter int64
		numGoroutines := 10
		errIdx := 5
		simulatedErr := fmt.Errorf("simulated error")

		for i := range numGoroutines {
			cg.Run(func() error {
				atomic.AddInt64(&counter, 1)
				if i == errIdx {
					return simulatedErr
				}
				return nil
			})
		}

		errs := cg.Wait()

		require.Len(t, errs, numGoroutines)

		require.Equal(t, int64(numGoroutines), atomic.LoadInt64(&counter))

		for i, err := range errs {
			if i == errIdx {
				require.ErrorIs(t, err, simulatedErr)
			} else {
				require.NoError(t, err)
			}
		}
	})

	t.Run("limited", func(t *testing.T) {
		const limit = 3
		const numGoroutines = 4

		ctx := WithConcurrencyLimit(context.Background(), limit)
		cg1 := NewConcurrencyGroup(ctx)
		cg2 := NewConcurrencyGroup(ctx)

		var activeCount int64
		var maxActive int64
		var mu sync.Mutex

		for range numGoroutines {
			cg1.Run(func() error {
				current := atomic.AddInt64(&activeCount, 1)

				mu.Lock()
				if current > maxActive {
					maxActive = current
				}
				mu.Unlock()

				time.Sleep(50 * time.Millisecond)
				atomic.AddInt64(&activeCount, -1)
				return nil
			})

			cg2.Run(func() error {
				current := atomic.AddInt64(&activeCount, 1)

				mu.Lock()
				if current > maxActive {
					maxActive = current
				}
				mu.Unlock()

				time.Sleep(50 * time.Millisecond)
				atomic.AddInt64(&activeCount, -1)
				return nil
			})
		}

		errs1 := cg1.Wait()
		require.Len(t, errs1, numGoroutines)
		for _, err := range errs1 {
			require.NoError(t, err)
		}

		errs2 := cg2.Wait()
		require.Len(t, errs2, numGoroutines)
		for _, err := range errs2 {
			require.NoError(t, err)
		}

		require.LessOrEqual(t, maxActive, int64(limit))
	})

	t.Run("wait", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			ctx := context.Background()
			cg := NewConcurrencyGroup(ctx)

			errs := cg.Wait()

			require.Empty(t, errs)
		})
		t.Run("multiple", func(t *testing.T) {
			ctx := context.Background()
			cg := NewConcurrencyGroup(ctx)

			cg.Run(func() error {
				return fmt.Errorf("test error")
			})

			errs1 := cg.Wait()
			require.Len(t, errs1, 1)

			errs2 := cg.Wait()
			require.Len(t, errs2, 1)

			require.Equal(t, errs1[0].Error(), errs2[0].Error())
		})
	})
}
