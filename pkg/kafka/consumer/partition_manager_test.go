package consumer

import (
	"context"
	"testing"

	consumerMocks "github.com/justtrackio/gosoline/pkg/kafka/consumer/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartitionManagerIgnoresAssignmentsAfterStop(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	messageHandler := consumerMocks.NewKafkaMessageHandler(t)

	messageHandler.EXPECT().Stop().Once()

	manager := NewPartitionManager(logger, messageHandler)
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
