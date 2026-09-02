package producer_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/justtrackio/gosoline/pkg/kafka/producer"
	kafkaProducerMocks "github.com/justtrackio/gosoline/pkg/kafka/producer/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestProducerProduceSyncSuccess(t *testing.T) {
	writer := kafkaProducerMocks.NewWriter(t)
	metricWriter := metricMocks.NewWriter(t)

	records := []*kgo.Record{{Value: []byte("test1")}, {Value: []byte("test2")}}
	results := kgo.ProduceResults{
		{Record: records[0], Err: nil},
		{Record: records[1], Err: nil},
	}

	writer.EXPECT().ProduceSync(mock.Anything, records[0], records[1]).Return(results)
	metricWriter.EXPECT().Write(mock.Anything, mock.MatchedBy(func(data metric.Data) bool {
		if len(data) != 3 {
			return false
		}

		return hasNoUnitOrKind(data) &&
			data[0].MetricName == "batch.size" && data[0].Value == 2.0 &&
			data[1].MetricName == "client.operation.duration" &&
			data[2].MetricName == "client.sent.messages" && data[2].Value == 2.0
	})).Once()

	p := producer.NewProducerWithInterfaces(writer, metricWriter, "test-producer", "test-topic")

	err := p.ProduceSync(context.Background(), records...)
	assert.NoError(t, err)
}

func TestProducerProduceSyncFailure(t *testing.T) {
	writer := kafkaProducerMocks.NewWriter(t)
	metricWriter := metricMocks.NewWriter(t)

	records := []*kgo.Record{{Value: []byte("test1")}, {Value: []byte("test2")}}
	results := kgo.ProduceResults{
		{Record: records[0], Err: nil},
		{Record: records[1], Err: errors.New("produce error")},
	}

	writer.EXPECT().ProduceSync(mock.Anything, records[0], records[1]).Return(results)
	metricWriter.EXPECT().Write(mock.Anything, mock.MatchedBy(func(data metric.Data) bool {
		if len(data) != 4 {
			return false
		}

		return hasNoUnitOrKind(data) &&
			data[0].MetricName == "batch.size" && data[0].Value == 2.0 &&
			data[1].MetricName == "client.operation.duration" &&
			data[2].MetricName == "client.sent.messages" && data[2].Value == 1.0 &&
			data[3].MetricName == "send.errors" && data[3].Value == 1.0
	})).Once()

	p := producer.NewProducerWithInterfaces(writer, metricWriter, "test-producer", "test-topic")

	err := p.ProduceSync(context.Background(), records...)
	assert.Error(t, err)
}

func TestProducerProduceSyncBatchSize(t *testing.T) {
	writer := kafkaProducerMocks.NewWriter(t)
	metricWriter := metricMocks.NewWriter(t)

	records := []*kgo.Record{{Value: []byte("a")}, {Value: []byte("b")}, {Value: []byte("c")}}
	results := kgo.ProduceResults{
		{Record: records[0], Err: nil},
		{Record: records[1], Err: nil},
		{Record: records[2], Err: nil},
	}

	writer.EXPECT().ProduceSync(mock.Anything, records[0], records[1], records[2]).Return(results)
	metricWriter.EXPECT().Write(mock.Anything, mock.MatchedBy(func(data metric.Data) bool {
		if len(data) != 3 {
			return false
		}

		return hasNoUnitOrKind(data) &&
			data[0].MetricName == "batch.size" && data[0].Value == 3.0 &&
			data[1].MetricName == "client.operation.duration" &&
			data[2].MetricName == "client.sent.messages" && data[2].Value == 3.0
	})).Once()

	p := producer.NewProducerWithInterfaces(writer, metricWriter, "test-producer", "test-topic")

	err := p.ProduceSync(context.Background(), records...)
	assert.NoError(t, err)
}

func hasNoUnitOrKind(data metric.Data) bool {
	for _, datum := range data {
		if datum.Unit != "" || !reflect.DeepEqual(datum.Kind, metric.Kind{}) {
			return false
		}
	}

	return true
}
