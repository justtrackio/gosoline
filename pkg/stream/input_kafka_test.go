package stream

import (
	"context"
	"testing"

	kafkaConsumerMocks "github.com/justtrackio/gosoline/pkg/kafka/consumer/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestKafkaInputRunDelegatesToConsumer(t *testing.T) {
	consumer := kafkaConsumerMocks.NewConsumer(t)

	consumer.EXPECT().Run(mock.Anything).Return(nil).Once()

	input := NewKafkaInputWithInterfaces(consumer, nil, make(chan *Message))

	err := input.Run(context.Background())
	assert.NoError(t, err)
}

func TestKafkaInputStopDelegatesToConsumer(t *testing.T) {
	consumer := kafkaConsumerMocks.NewConsumer(t)

	consumer.EXPECT().Stop(mock.Anything).Once()

	input := NewKafkaInputWithInterfaces(consumer, nil, make(chan *Message))

	input.Stop(context.Background())
}

func TestKafkaInputIsHealthy(t *testing.T) {
	consumer := kafkaConsumerMocks.NewConsumer(t)

	consumer.EXPECT().IsHealthy().Return(true).Once()

	input := NewKafkaInputWithInterfaces(consumer, nil, make(chan *Message))

	assert.True(t, input.IsHealthy())
}

func TestKafkaInputIsUnhealthyWhenConsumerUnhealthy(t *testing.T) {
	consumer := kafkaConsumerMocks.NewConsumer(t)

	consumer.EXPECT().IsHealthy().Return(false).Once()

	input := NewKafkaInputWithInterfaces(consumer, nil, make(chan *Message))

	assert.False(t, input.IsHealthy())
}

func TestKafkaInputIsUnhealthyWhenSchemaRegistryNotReady(t *testing.T) {
	consumer := kafkaConsumerMocks.NewConsumer(t)

	consumer.EXPECT().IsHealthy().Return(true).Once()

	inp := NewKafkaInputWithInterfaces(consumer, nil, make(chan *Message)).(*kafkaInput)
	inp.schemaRegistryReady.Store(false)

	assert.False(t, inp.IsHealthy())
}
