package consumer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

// testMessageHandler is a hand written stub instead of the generated mock: this test lives in the consumer package
// itself and the generated mocks import the consumer package, which would create an import cycle.
type testMessageHandler struct {
	stopped atomic.Int32
}

func (h *testMessageHandler) Handle(messages []*kgo.Record) BatchCompletion {
	done := make(chan struct{})
	close(done)

	return testBatchCompletion{done: done}
}

func (h *testMessageHandler) Stop() {
	h.stopped.Add(1)
}

type testBatchCompletion struct {
	done chan struct{}
}

func (c testBatchCompletion) Done() <-chan struct{} { return c.done }
func (c testBatchCompletion) FailedCount() int      { return 0 }

func TestPartitionManagerIgnoresAssignmentsAfterStop(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	messageHandler := &testMessageHandler{}
	metricWriter := metricMocks.NewWriter(t)

	manager := NewPartitionManager(logger, clock.NewFakeClock(), metricWriter, messageHandler, time.Second, "test-consumer")
	manager.Stop(context.Background())

	assert.Equal(t, int32(1), messageHandler.stopped.Load())

	require.NotPanics(t, func() {
		manager.OnPartitionsAssigned(context.Background(), nil, map[string][]int32{
			"topic": {1},
		})
	})

	manager.lck.RLock()
	defer manager.lck.RUnlock()

	assert.Empty(t, manager.consumers)
}
