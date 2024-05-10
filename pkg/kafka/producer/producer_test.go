package producer_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/kafka/producer"
	producerMocks "github.com/justtrackio/gosoline/pkg/kafka/producer/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_Write_WriteOne(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(t.Context())
		logger      = logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
		writer      = producerMocks.NewWriter(t)
		conf        = &producer.Settings{
			FQTopic: "fq-topic",
		}
		messages = []kafka.Message{
			{
				Partition: 0,
				Offset:    1,
				Key:       []byte("0"),
				Value:     []byte("0"),
				Headers:   []protocol.Header{},
			},
			{
				Partition: 1,
				Offset:    2,
				Key:       []byte("1"),
				Value:     []byte("1"),
				Headers:   []protocol.Header{},
			},
			{
				Partition: 2,
				Offset:    3,
				Key:       []byte("2"),
				Value:     []byte("2"),
				Headers:   []protocol.Header{},
			},
			{
				Partition: 3,
				Offset:    4,
				Key:       []byte("3"),
				Value:     []byte("3"),
				Headers:   []protocol.Header{},
			},
		}
	)

	writer.On("WriteMessages", matcher.Context, mock.AnythingOfType("[]kafka.Message")).Return(
		func(ctx context.Context, ms ...kafka.Message) error {
			dead, _ := ctx.Deadline()
			assert.Less(t, dead, time.Now().Add(time.Minute))

			// Write()
			for i := 0; i < len(messages); i++ {
				assert.Equal(
					t,
					ms[i],
					kafka.Message{
						Headers: messages[i].Headers,
						Key:     messages[i].Key,
						Value:   messages[i].Value,
						Topic:   conf.FQTopic,
					},
				)
			}

			return nil
		},
	).Times(1)

	writer.On("WriteMessages", matcher.Context, mock.AnythingOfType("[]kafka.Message")).Return(
		func(ctx context.Context, ms ...kafka.Message) error {
			var (
				i       = 0
				dead, _ = ctx.Deadline()
			)
			assert.Less(t, dead, time.Now().Add(time.Minute))

			// WriteOne()
			assert.Len(t, ms, 1)

			assert.Equal(t,
				ms[i],
				kafka.Message{
					Headers: messages[i].Headers,
					Key:     messages[i].Key,
					Value:   messages[i].Value,
					Topic:   conf.FQTopic,
				})

			return nil
		},
	).Times(1)

	prod, err := producer.NewProducerWithInterfaces(conf, logger, writer)
	assert.Nil(t, err)

	// Data should contain batch size of messages after consumer is started.
	go func() {
		err = prod.Run(ctx)
		assert.Error(t, err)
	}()

	// Data should be written to the writer.
	err = prod.Write(ctx, messages...)
	assert.Nil(t, err)

	err = prod.WriteOne(ctx, messages[0])
	assert.Nil(t, err)

	// Should flush messages on shutdown.
	writer.On("Close").Times(1)
	cancel()
	time.Sleep(time.Second)
}
