package consumer

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/clock"
	kafkaConsumerMocks "github.com/justtrackio/gosoline/pkg/kafka/consumer/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartitionManagerIgnoresAssignmentsAfterStop(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	messageHandler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)

	messageHandler.EXPECT().Stop().Once()

	manager := NewPartitionManager(logger, clock.NewFakeClock(), metricWriter, messageHandler, "test-consumer")
	manager.Stop(context.Background())

	require.NotPanics(t, func() {
		manager.OnPartitionsAssigned(context.Background(), nil, map[string][]int32{
			"topic": {1},
		})
	})

	manager.lck.RLock()
	defer manager.lck.RUnlock()

	assert.Empty(t, manager.consumers)
}
