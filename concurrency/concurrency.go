package concurrency

import (
	"context"
	"sync"
)

type limiterKey struct{}

type limiterType struct {
	limit uint
	ch    chan struct{}
}

func newLimiter(limit uint) *limiterType {
	return &limiterType{
		limit: limit,
		ch:    make(chan struct{}, limit),
	}
}

func (l *limiterType) Limit() uint {
	return l.limit
}

func (l *limiterType) Add() {
	l.ch <- struct{}{}
}

func (l *limiterType) Done() {
	<-l.ch
}

// WithConcurrencyLimit returns a new context with the given concurrency limit embedded.
// If the parent context already has a limit, this new limit overrides it.
func WithConcurrencyLimit(ctx context.Context, limit uint) context.Context {
	return context.WithValue(ctx, limiterKey{}, newLimiter(limit))
}

func getLimiter(ctx context.Context) *limiterType {
	limiter, hasLimit := ctx.Value(limiterKey{}).(*limiterType)
	if hasLimit {
		return limiter
	}
	return nil
}

// ConcurrencyGroup manages running concurrent go routines respecting a context limit defined by
// WithConcurrencyLimit. This limit is global and shared by all ConcurrencyGroup. For example, if
// the context limit is 2, and we have 2 ConcurrencyGroup attempting to run 5 go routines each,
// all go routines from both ConcurrencyGroup shares the same global limit of 2. This means that,
// 2 go routines can run concurrently (from any ConcurrencyGroup), and all other 8, will have to
// wait for one of the first 2 to complete.
type ConcurrencyGroup struct {
	context context.Context
	wg      sync.WaitGroup
	errs    []error
	errsMu  sync.Mutex
}

func NewConcurrencyGroup(ctx context.Context) *ConcurrencyGroup {
	return &ConcurrencyGroup{
		context: ctx,
	}
}

// Run adds given function to be executed as a go routine and runs it, as soon as the context limit
// allows.
func (c *ConcurrencyGroup) Run(fn func() error) {
	limiter := getLimiter(c.context)
	if limiter != nil {
		limiter.Add()
	}

	c.errsMu.Lock()
	i := len(c.errs)
	c.errs = append(c.errs, nil)
	c.errsMu.Unlock()

	c.wg.Add(1)
	go func() {
		if limiter != nil {
			defer limiter.Done()
		}
		defer c.wg.Done()
		err := fn()
		c.errsMu.Lock()
		c.errs[i] = err
		c.errsMu.Unlock()
	}()
}

// Wait waits for all active go routines to complete, and uses errors.Join to concatenate errors
// from all of them.
func (c *ConcurrencyGroup) Wait() []error {
	c.wg.Wait()
	return c.errs
}

// BatchRun divides the items slice into batches and runs the provided function on each batch
// concurrently. The number of batches is determined by the concurrency limit in the context, set by
// WithConcurrencyLimit. If no limit is set, each item will be processed in its own batch. Items
// are distributed as evenly as possible across batches.
func BatchRun[I any](ctx context.Context, items []I, fn func([]I) error) []error {
	var maxBatches int

	limiter := getLimiter(ctx)
	if limiter != nil {
		maxBatches = int(limiter.limit)
	} else {
		maxBatches = len(items)
	}

	batchCount := min(len(items), maxBatches)
	batches := make([][]I, batchCount)

	baseSize := len(items) / batchCount
	remainder := len(items) % batchCount

	startIdx := 0
	for i := range batchCount {
		batchSize := baseSize
		if i < remainder {
			batchSize++
		}
		endIdx := startIdx + batchSize
		batches[i] = items[startIdx:endIdx]
		startIdx = endIdx
	}

	concurrencyGroup := NewConcurrencyGroup(ctx)

	for _, batch := range batches {
		concurrencyGroup.Run(func() error { return fn(batch) })
	}

	return concurrencyGroup.Wait()
}
