package producer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	metricNameRecordsSent       = "client.sent.messages"
	metricNameRecordsSentFailed = "send.errors"
	metricNameProduceBatchSize  = "batch.size"
	metricNameProduceDuration   = "client.operation.duration"
)

//go:generate go run github.com/vektra/mockery/v2 --name Producer
type Producer interface {
	ProduceSync(ctx context.Context, records ...*kgo.Record) error
}

type producer struct {
	writer       Writer
	clock        clock.Clock
	metricWriter metric.Writer
	name         string
	topicName    string
}

func NewProducer(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings, name string) (Producer, error) {
	writer, err := NewWriter(ctx, config, logger, settings, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka writer: %w", err)
	}

	fullTopicName, err := kafka.BuildFullTopicName(config, settings.ToIdentity(), settings.TopicId)
	if err != nil {
		return nil, fmt.Errorf("failed to build full topic name for topic id %q: %w", settings.TopicId, err)
	}

	conn, err := connection.ParseSettings(config, settings.Connection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", settings.Connection, err)
	}

	if err = reslife.AddLifeCycleer(ctx, NewLifecycleManagerProducer(name, fullTopicName, conn.Brokers)); err != nil {
		return nil, fmt.Errorf("failed to add kafka producer lifecycle manager: %w", err)
	}

	defaults := getProducerDefaultMetrics(name, fullTopicName)
	metricWriter := metric.NewWriter(metric.NamespaceKafkaProducer, defaults...)

	return NewProducerWithInterfaces(writer, metricWriter, name, fullTopicName), nil
}

func NewProducerWithInterfaces(writer Writer, metricWriter metric.Writer, name, topicName string) Producer {
	return &producer{
		writer:       writer,
		clock:        clock.Provider,
		metricWriter: metricWriter,
		name:         name,
		topicName:    topicName,
	}
}

func (p *producer) ProduceSync(ctx context.Context, records ...*kgo.Record) error {
	start := p.clock.Now()
	results := p.writer.ProduceSync(ctx, records...)
	durationMs := float64(p.clock.Since(start).Milliseconds())

	dims := metric.Dimensions{kafka.DimensionClientType: kafka.ClientTypeProducer, kafka.DimensionClient: p.name, kafka.DimensionTopic: p.topicName}

	data := metric.Data{
		{Priority: metric.PriorityHigh, MetricName: metricNameProduceBatchSize, Dimensions: dims, Value: float64(len(records))},
		{Priority: metric.PriorityHigh, Namespace: metric.NamespaceMessaging, MetricName: metricNameProduceDuration, Dimensions: dims, Value: durationMs},
	}

	if err := results.FirstErr(); err != nil {
		var sent, failed int
		for _, r := range results {
			if r.Err != nil {
				failed++
			} else {
				sent++
			}
		}

		data = append(data,
			recordsSentDatum(dims, float64(sent)),
			&metric.Datum{Priority: metric.PriorityHigh, MetricName: metricNameRecordsSentFailed, Dimensions: dims, Value: float64(failed)},
		)

		p.metricWriter.Write(ctx, data)

		return err
	}

	data = append(data, recordsSentDatum(dims, float64(len(records))))

	p.metricWriter.Write(ctx, data)

	return nil
}

// recordsSentDatum reports messages handed to the broker under the messaging semantic convention.
func recordsSentDatum(dims metric.Dimensions, value float64) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		Namespace:  metric.NamespaceMessaging,
		MetricName: metricNameRecordsSent,
		Dimensions: dims,
		Value:      value,
	}
}

func getProducerDefaultMetrics(name, topicName string) metric.Data {
	dims := metric.Dimensions{kafka.DimensionClientType: kafka.ClientTypeProducer, kafka.DimensionClient: name, kafka.DimensionTopic: topicName}

	return metric.Data{
		{Priority: metric.PriorityHigh, Namespace: metric.NamespaceMessaging, MetricName: metricNameRecordsSent, Dimensions: dims, Unit: metric.UnitCount, Kind: metric.KindCounter.Build()},
		{Priority: metric.PriorityHigh, MetricName: metricNameRecordsSentFailed, Dimensions: dims, Unit: metric.UnitCount, Kind: metric.KindCounter.Build()},
		{Priority: metric.PriorityHigh, MetricName: metricNameProduceBatchSize, Dimensions: dims, Unit: metric.UnitCountAverage, Kind: metric.KindHistogram.Build()},
		{Priority: metric.PriorityHigh, Namespace: metric.NamespaceMessaging, MetricName: metricNameProduceDuration, Dimensions: dims, Unit: metric.UnitMillisecondsAverage, Kind: metric.KindHistogram.Build()},
	}
}
