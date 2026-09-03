package consumer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	metricNameProcessDuration       = "process.duration"
	metricNameWaitDuration          = "wait.duration"
	metricNameCommitDuration        = "commit.duration"
	metricNameCommitFailures        = "commit.errors"
	metricNameRecordsConsumedFailed = "consume.errors"
)

type PartitionConsumer struct {
	logger         log.Logger
	clock          clock.Clock
	metricWriter   metric.Writer
	name           string
	topic          string
	partition      int32
	messageHandler KafkaMessageHandler
	kafkaClient    *kgo.Client
	assignedBatch  chan []*kgo.Record
	stop           chan struct{}
	done           chan struct{}
}

func NewPartitionConsumer(logger log.Logger, clk clock.Clock, metricWriter metric.Writer, messageHandler KafkaMessageHandler, kafkaClient *kgo.Client, name, topic string, partition int32) *PartitionConsumer {
	return &PartitionConsumer{
		logger:         logger,
		clock:          clk,
		metricWriter:   metricWriter,
		name:           name,
		topic:          topic,
		partition:      partition,
		messageHandler: messageHandler,
		kafkaClient:    kafkaClient,
		assignedBatch:  make(chan []*kgo.Record),
		stop:           make(chan struct{}),
		done:           make(chan struct{}),
	}
}

func (c *PartitionConsumer) Consume(ctx context.Context) error {
	defer c.logger.Debug(ctx, "done consuming partition %d of topic %s", c.partition, c.topic)
	defer close(c.done)

	waitStart := c.clock.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stop:
			return nil
		case records := <-c.assignedBatch:
			if len(records) == 0 {
				continue
			}

			waitMs := float64(c.clock.Since(waitStart).Milliseconds())

			processStart := c.clock.Now()
			handleFailed := c.handleWithRecovery(ctx, records)
			processMs := float64(c.clock.Since(processStart).Milliseconds())

			commitStart := c.clock.Now()
			err := c.kafkaClient.CommitRecords(ctx, records...)
			commitMs := float64(c.clock.Since(commitStart).Milliseconds())

			var data metric.Data
			data = append(data, c.metricPair(metricNamespaceKafkaConsumer, metricNameWaitDuration, waitMs, metric.UnitMillisecondsAverage, metric.KindHistogram.Build())...)
			data = append(data, c.metricPair(metricNamespaceMessaging, metricNameProcessDuration, processMs, metric.UnitMillisecondsAverage, metric.KindHistogram.Build())...)
			data = append(data, c.metricPair(metricNamespaceKafkaConsumer, metricNameCommitDuration, commitMs, metric.UnitMillisecondsAverage, metric.KindHistogram.Build())...)

			if err != nil {
				data = append(data, c.metricPair(metricNamespaceKafkaConsumer, metricNameCommitFailures, 1.0, metric.UnitCount, metric.KindCounter.Build())...)

				c.metricWriter.Write(ctx, data)

				offset := records[len(records)-1].Offset + 1

				return fmt.Errorf("failed to commit offset %d for partition %d of topic %s: %w", offset, c.partition, c.topic, err)
			}

			if handleFailed {
				data = append(data, c.metricPair(metricNamespaceKafkaConsumer, metricNameRecordsConsumedFailed, float64(len(records)), metric.UnitCount, metric.KindCounter.Build())...)
			}

			c.metricWriter.Write(ctx, data)
			waitStart = c.clock.Now()
		}
	}
}

// metricPair reports one measurement of this partition consumer at both topic and partition
// granularity.
func (c *PartitionConsumer) metricPair(namespace string, name string, value float64, unit metric.StandardUnit, metricKind metric.Kind) metric.Data {
	return kafka.MetricPair(kafka.MetricSpec{
		ClientType: kafka.ClientTypeConsumer,
		ClientName: c.name,
		Namespace:  namespace,
		Name:       name,
		Topic:      c.topic,
		Partition:  c.partition,
		Value:      value,
		Unit:       unit,
		Kind:       metricKind,
	})
}

func (c *PartitionConsumer) handleWithRecovery(ctx context.Context, records []*kgo.Record) (failed bool) {
	defer func() {
		if err := coffin.ResolveRecovery(recover()); err != nil {
			c.logger.Error(ctx, "panic in message handler for partition %d of topic %s: %w", c.partition, c.topic, err)
			failed = true
		}
	}()

	c.messageHandler.Handle(records)

	return false
}

func (c *PartitionConsumer) Stop() {
	close(c.stop)
}
