package producer

import (
	"context"
	"fmt"
	"maps"

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
	metricNamespaceKafka         = "kafka"
	metricNamespaceKafkaProducer = "kafka.producer"

	metricNameRecordsSent      = "sent.messages"
	metricNameProduceBatchSize = "batch.records"
	metricNameProduceDuration  = "produce.duration"
)

func init() {
	metric.RegisterHelp(metricNamespaceKafkaProducer, metricNameProduceBatchSize, "records a kafka producer sent per batch")
	metric.RegisterHelp(metricNamespaceKafkaProducer, metricNameRecordsSent, "records a kafka producer handed to the broker, by error type")
	metric.RegisterHelp(metricNamespaceKafkaProducer, metricNameProduceDuration, "duration of a kafka producer send")
}

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
	metricWriter := metric.NewWriter(metricNamespaceKafkaProducer, defaults...)

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
		{Priority: metric.PriorityHigh, Namespace: metricNamespaceKafkaProducer, MetricName: metricNameProduceDuration, Dimensions: dims, Value: durationMs},
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
			recordsSentDatum(dims, metric.DimensionDefault, float64(sent)),
			recordsSentDatum(dims, metric.ErrorType(err), float64(failed)),
		)

		p.metricWriter.Write(ctx, data)

		return err
	}

	data = append(data, recordsSentDatum(dims, metric.DimensionDefault, float64(len(records))))

	p.metricWriter.Write(ctx, data)

	return nil
}

// recordsSentDatum reports records handed to the broker. A record that failed to send is the same
// metric told apart by its error type, so a failure needs no metric of its own.
func recordsSentDatum(dims metric.Dimensions, errorType string, value float64) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		Namespace:  metricNamespaceKafkaProducer,
		MetricName: metricNameRecordsSent,
		Dimensions: withErrorType(dims, errorType),
		Value:      value,
	}
}

// withErrorType copies dims and adds the error type, so the caller's map is never mutated.
func withErrorType(dims metric.Dimensions, errorType string) metric.Dimensions {
	out := make(metric.Dimensions, len(dims)+1)
	maps.Copy(out, dims)
	out[metric.DimensionErrorType] = errorType

	return out
}

func getProducerDefaultMetrics(name, topicName string) metric.Data {
	dims := metric.Dimensions{kafka.DimensionClientType: kafka.ClientTypeProducer, kafka.DimensionClient: name, kafka.DimensionTopic: topicName}

	return metric.Data{
		{Priority: metric.PriorityHigh, Namespace: metricNamespaceKafkaProducer, MetricName: metricNameRecordsSent, Dimensions: withErrorType(dims, metric.DimensionDefault), Unit: metric.UnitCount, Kind: metric.KindCounter.Build()},
		{Priority: metric.PriorityHigh, MetricName: metricNameProduceBatchSize, Dimensions: dims, Unit: metric.UnitCountAverage, Kind: metric.KindHistogram.Build()},
		{Priority: metric.PriorityHigh, Namespace: metricNamespaceKafkaProducer, MetricName: metricNameProduceDuration, Dimensions: dims, Unit: metric.UnitMillisecondsAverage, Kind: metric.KindHistogram.Build()},
	}
}
