//go:generate minimock -i requestDeduplicator -o ./mock/ -s ".go" -g

package containersmap

import (
	"context"
	"fmt"
	"sync"

	"github.com/Snyssfx/container_scheduler/internal/deduplicator"
	"go.uber.org/zap"
)

// ContainersMap is a map of seeds to containers.
// Containers are called deduplicators because they hold the logic to
// deduplicate several user requests into one calculation.
type ContainersMap struct {
	l *zap.SugaredLogger

	mu                 sync.RWMutex
	seedToDeduplicator map[int]requestDeduplicator
	// TODO: add limit of maximum containers count
	// TODO: add a function that is a fabric of deduplicators. It is needed for dependency inversion.
}

type requestDeduplicator interface {
	Calculate(ctx context.Context, input int) (int, error)
	Close() error
}

// New creates new ContainersMap
func New(logger *zap.SugaredLogger) *ContainersMap {
	return &ContainersMap{
		l:                  logger,
		mu:                 sync.RWMutex{},
		seedToDeduplicator: make(map[int]requestDeduplicator),
	}
}

// Close closes all deduplicators.
func (c *ContainersMap) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, d := range c.seedToDeduplicator {
		err := d.Close()
		if err != nil {
			return fmt.Errorf("cannot close deduplicator: %w", err)
		}
	}

	c.l.Infof("container map closed")
	return nil
}

// Calculate gets existing or creates new deduplicator and he calculates a result.
func (c *ContainersMap) Calculate(ctx context.Context, seed, input int) (int, error) {
	d, err := c.getOrCreateDeduplicator(seed)
	if err != nil {
		return 0, err
	}

	return d.Calculate(ctx, input)
}

func (c *ContainersMap) getOrCreateDeduplicator(seed int) (requestDeduplicator, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var d requestDeduplicator
	var ok bool
	var err error

	d, ok = c.seedToDeduplicator[seed]
	if !ok {
		d, err = deduplicator.NewCachedDeduplicator(c.l.Named("cached"), seed)
		if err != nil {
			return nil, fmt.Errorf("cannot create cached deduplicator: %w", err)
		}

		c.seedToDeduplicator[seed] = d
		c.l.Infof("container %d created", seed)
	}

	return d, nil
}
