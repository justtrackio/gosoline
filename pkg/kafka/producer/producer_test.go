package producer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/kafka/producer"
	kafkaProducerMocks "github.com/justtrackio/gosoline/pkg/kafka/producer/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
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

		return data[0].MetricName == "batch.records" && data[0].Value == 2.0 &&
			data[1].MetricName == "produce.duration" &&
			data[2].MetricName == "sent.messages" && data[2].Value == 2.0
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

		return data[0].MetricName == "batch.records" && data[0].Value == 2.0 &&
			data[1].MetricName == "produce.duration" &&
			data[2].MetricName == "sent.messages" && data[2].Value == 1.0 &&
			data[2].Dimensions["error.type"] == "{{default}}" &&
			data[3].MetricName == "sent.messages" && data[3].Value == 1.0 &&
			data[3].Dimensions["error.type"] != "{{default}}"
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

		return data[0].MetricName == "batch.records" && data[0].Value == 3.0 &&
			data[1].MetricName == "produce.duration" &&
			data[2].MetricName == "sent.messages" && data[2].Value == 3.0
	})).Once()

	p := producer.NewProducerWithInterfaces(writer, metricWriter, "test-producer", "test-topic")

	err := p.ProduceSync(context.Background(), records...)
	assert.NoError(t, err)
}

func TestProducerProduceSyncEmitsFailedRecordsAsCounter(t *testing.T) {
	writer := kafkaProducerMocks.NewWriter(t)
	registry := prometheus.NewRegistry()
	metricWriter := metric.NewPrometheusWriterWithInterfaces(log.NewLogger(), registry, 1000, time.Second)

	records := []*kgo.Record{{Value: []byte("test")}}
	results := kgo.ProduceResults{
		{Record: records[0], Err: errors.New("produce error")},
	}
	writer.EXPECT().ProduceSync(mock.Anything, records[0]).Return(results)

	p := producer.NewProducerWithInterfaces(writer, metricWriter, "test-producer", "test-topic")
	err := p.ProduceSync(t.Context(), records...)
	assert.Error(t, err)

	families, err := registry.Gather()
	assert.NoError(t, err)
	var sent *dto.MetricFamily
	for _, family := range families {
		if family.GetName() == "gosoline_kafka_producer_sent_messages_total" {
			sent = family
			break
		}
	}
	if !assert.NotNil(t, sent) {
		return
	}

	assert.Equal(t, dto.MetricType_COUNTER, sent.GetType())
	var foundFailure bool
	var failureCount float64
	for _, sample := range sent.Metric {
		labels := make(map[string]string, len(sample.Label))
		for _, label := range sample.Label {
			labels[label.GetName()] = label.GetValue()
		}
		if labels["error_type"] == "send_failed" {
			foundFailure = true
			failureCount = sample.GetCounter().GetValue()
		}
	}
	assert.True(t, foundFailure)
	assert.Equal(t, 1.0, failureCount)
}
