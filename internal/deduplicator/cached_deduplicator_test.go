package deduplicator

import (
	"context"
	"sync"
	"testing"

	"github.com/Snyssfx/container_scheduler/internal/deduplicator/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCachedDeduplicator_Calculate(t *testing.T) {
	var nCalls int
	d := mock.NewRequestDeduplicatorMock(t)
	d.CalculateMock.Set(func(ctx context.Context, input int) (i1 int, err error) {
		nCalls++
		return 2, nil
	})
	cd := &CachedDeduplicator{
		l:             zap.NewNop().Sugar(),
		d:             d,
		mu:            sync.RWMutex{},
		inputToResult: map[int]int{},
	}

	got, err := cd.Calculate(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 2, got)
	got, err = cd.Calculate(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 2, got)

	assert.Equal(t, nCalls, 1)
}
