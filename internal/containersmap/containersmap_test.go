package containersmap

import (
	"context"
	"testing"

	"github.com/Snyssfx/container_scheduler/internal/containersmap/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestContainersMap_Calculate(t *testing.T) {
	c := New(zap.NewNop().Sugar(), func(l *zap.SugaredLogger, seed int) (RequestDeduplicator, error) {
		rd := mock.NewRequestDeduplicatorMock(t)
		rd.CalculateMock.Set(func(ctx context.Context, input int) (i1 int, err error) {
			assert.Equal(t, input, 1)
			return 2, nil
		})
		return rd, nil
	})

	_, err := c.Calculate(context.Background(), 1, 1)
	require.NoError(t, err)
	_, err = c.Calculate(context.Background(), 2, 1)
	require.NoError(t, err)
	_, err = c.Calculate(context.Background(), 1, 1)
	require.NoError(t, err)

	assert.Len(t, c.seedToDeduplicator, 2)
}
