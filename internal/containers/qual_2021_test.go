//go:generate minimock -i container -o ./mock/ -s ".go" -g
//go:generate minimock -i client -o ./mock/ -s ".go" -g

package containers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Snyssfx/container_scheduler/internal/containers/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestQual_Calculate(t *testing.T) {
	d := mock.NewContainerMock(t)
	d.RunMock.Return(nil)
	client := mock.NewClientMock(t)
	client.GetMock.Set(func(string) (rp1 *http.Response, err error) {
		return &http.Response{StatusCode: 200}, nil
	})
	client.DoMock.Set(func(rp1 *http.Request) (rp2 *http.Response, err error) {
		return &http.Response{
			Body: io.NopCloser(bytes.NewReader([]byte(`2`))),
		}, nil
	})
	q := &Qual{
		l:               zap.NewNop().Sugar(),
		d:               d,
		port:            9090,
		name:            "qual_9090_seed_123",
		client:          client,
		closeFn:         nil,
		stateMu:         sync.Mutex{},
		state:           initState,
		lastCalculation: time.Time{},
	}

	got, err := q.Calculate(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, readyState, q.state)
	assert.Equal(t, 2, got)
}

func TestQual_stopAfter(t *testing.T) {
	d := mock.NewContainerMock(t)
	d.StopMock.Return(nil)
	q := &Qual{
		l:               zap.L().Sugar(),
		d:               d,
		port:            9090,
		name:            "qual_9090_seed_123",
		client:          nil,
		closeFn:         nil,
		stateMu:         sync.Mutex{},
		state:           readyState,
		lastCalculation: time.Now().Add(-1 * time.Second),
	}

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	q.stopAfter(ctx, 1*time.Microsecond)
}
