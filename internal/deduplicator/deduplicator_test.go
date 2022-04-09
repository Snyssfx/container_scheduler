//go:generate minimock -i container -o ./mock/ -s ".go" -g

package deduplicator

import (
	"context"
	"sync"
	"testing"

	"github.com/Snyssfx/container_scheduler/internal/deduplicator/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

func TestRequestDeduplicator_Calculate_1Request(t *testing.T) {
	r, closeFn := newTestDeduplicator(t, 1)
	defer closeFn()

	res, err := r.Calculate(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, 1, res)
}

func TestRequestDeduplicator_Calculate_1000Requests(t *testing.T) {
	r, closeFn := newTestDeduplicator(t, 1)
	defer closeFn()

	wg := sync.WaitGroup{}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		i := i
		go func() {
			res, err := r.Calculate(context.Background(), i%5)

			require.NoError(t, err)
			assert.Equal(t, 1, res)
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestRequestDeduplicator_Calculate_1000RequestsWithUsubscribes(t *testing.T) {
	r, closeFn := newTestDeduplicator(t, 1)
	defer closeFn()

	wg := sync.WaitGroup{}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()

			ctx := context.Background()
			if i%2 == 0 {
				var cancelFn context.CancelFunc
				ctx, cancelFn = context.WithCancel(context.Background())
				cancelFn()
			}

			res, err := r.Calculate(ctx, 1)

			if i%2 != 0 {
				require.NoError(t, err)
				assert.Equal(t, 1, res)
			}
		}()
	}

	wg.Wait()
}

func TestRequestDeduplicator_Calculate_1000RequestsAllUsubscribes(t *testing.T) {
	r, closeFn := newTestDeduplicator(t, 1)
	defer closeFn()

	wg := sync.WaitGroup{}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancelFn := context.WithCancel(context.Background())
			cancelFn()
			_, err := r.Calculate(ctx, 1)

			require.Error(t, err)
			require.Contains(t, err.Error(), "request was canceled")
		}()
	}

	wg.Wait()
}

func newTestDeduplicator(t *testing.T, result int) (*RequestDeduplicator, context.CancelFunc) {
	t.Helper()

	c := mock.NewContainerMock(t)
	c.CalculateMock.Set(func(ctx context.Context, input int) (i1 int, err error) {
		return result, nil
	})
	ctx, cancelFn := context.WithCancel(context.Background())
	r := &RequestDeduplicator{
		l:                   zap.L().Sugar(),
		seed:                1,
		container:           c,
		reqID:               atomic.NewInt64(0),
		closeCtx:            ctx,
		closeLoopFn:         cancelFn,
		signalOfNewSub:      make(chan bool, 1),
		mu:                  sync.Mutex{},
		inputToSubsriptions: make(map[int]map[int]*subscription),
		curInput:            0,
		cancelCurCalcFn:     nil,
	}
	go r.Start()

	return r, cancelFn
}
