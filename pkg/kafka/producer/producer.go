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
	metricNameRecordsSent       = "RecordsSent"
	metricNameRecordsSentFailed = "RecordsSentFailed"
	metricNameProduceBatchSize  = "ProduceBatchSize"
	metricNameProduceDuration   = "ProduceDuration"
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
	metricWriter := metric.NewWriter(defaults...)

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

	dims := metric.Dimensions{kafka.DimensionProducer: p.name, kafka.DimensionTopic: p.topicName}

	data := metric.Data{
		metric.NewMetricDatum(metricNameProduceBatchSize, dims, float64(len(records)), metric.UnitCountAverage, metric.PriorityHigh),
		metric.NewMetricDatum(metricNameProduceDuration, dims, durationMs, metric.UnitMillisecondsAverage, metric.PriorityHigh),
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
			metric.NewMetricDatum(metricNameRecordsSent, dims, float64(sent), metric.UnitCount, metric.PriorityHigh),
			metric.NewMetricDatum(metricNameRecordsSentFailed, dims, float64(failed), metric.UnitCount, metric.PriorityHigh),
		)

		p.metricWriter.Write(ctx, data)

		return err
	}

	data = append(data, metric.NewMetricDatum(metricNameRecordsSent, dims, float64(len(records)), metric.UnitCount, metric.PriorityHigh))

	p.metricWriter.Write(ctx, data)

	return nil
}

func getProducerDefaultMetrics(name, topicName string) metric.Data {
	dims := metric.Dimensions{kafka.DimensionProducer: name, kafka.DimensionTopic: topicName}

	return metric.Data{
		{Priority: metric.PriorityHigh, MetricName: metricNameRecordsSent, Dimensions: dims, Unit: metric.UnitCount, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameRecordsSentFailed, Dimensions: dims, Unit: metric.UnitCount, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameProduceBatchSize, Dimensions: dims, Unit: metric.UnitCountAverage, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameProduceDuration, Dimensions: dims, Unit: metric.UnitMillisecondsAverage, Kind: metric.KindDefault},
	}
}
