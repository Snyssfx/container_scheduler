package deduplicator

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// CachedDeduplicator is a middleware between containersMap and RequestDeduplicator.
// It caches all results from a RequestDeduplicator.
type CachedDeduplicator struct {
	l *zap.SugaredLogger
	d *RequestDeduplicator

	// TODO: add hard limits and eviction for a cache.
	mu            sync.RWMutex
	inputToResult map[int]int
}

// NewCachedDeduplicator creates CachedDeduplicator.
func NewCachedDeduplicator(l *zap.SugaredLogger, seed int) (*CachedDeduplicator, error) {
	d, err := NewRequestDeduplicator(l.Named("dp"), seed)
	if err != nil {
		return nil, fmt.Errorf("cannot create deduplicator: %w", err)
	}

	go d.Start()

	return &CachedDeduplicator{
		l: l, d: d,
		mu:            sync.RWMutex{},
		inputToResult: make(map[int]int),
	}, nil
}

// Calculate gets the result from cache or calls RequestDeduplicator.Calculate.
func (cd *CachedDeduplicator) Calculate(ctx context.Context, input int) (int, error) {
	cd.mu.RLock()
	if res, ok := cd.inputToResult[input]; ok {
		cd.mu.RUnlock()
		cd.l.Infof("input %d, got result from cache: %d", input, res)
		return res, nil
	}
	cd.mu.RUnlock()

	res, err := cd.d.Calculate(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("cannot get res from requestDedulpicator %d: %w", input, err)
	}

	cd.mu.Lock()
	defer cd.mu.Unlock()
	cd.inputToResult[input] = res

	cd.l.Infof("saved res %d for input %d to a cache", res, input)
	return res, nil
}

// Close closes underlying RequestDeduplicator.
func (cd *CachedDeduplicator) Close() error {
	return cd.d.Close()
}
