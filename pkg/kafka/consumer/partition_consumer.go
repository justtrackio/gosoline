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
	metricNameProcessDuration       = "ProcessDuration"
	metricNameWaitDuration          = "WaitDuration"
	metricNameCommitDuration        = "CommitDuration"
	metricNameCommitFailures        = "CommitFailures"
	metricNameRecordsConsumedFailed = "RecordsConsumedFailed"
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
			data = append(data, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameWaitDuration, c.topic, c.partition, waitMs, metric.UnitMillisecondsAverage)...)
			data = append(data, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameProcessDuration, c.topic, c.partition, processMs, metric.UnitMillisecondsAverage)...)
			data = append(data, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameCommitDuration, c.topic, c.partition, commitMs, metric.UnitMillisecondsAverage)...)

			if err != nil {
				data = append(data, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameCommitFailures, c.topic, c.partition, 1.0, metric.UnitCount)...)

				c.metricWriter.Write(ctx, data)

				offset := records[len(records)-1].Offset + 1

				return fmt.Errorf("failed to commit offset %d for partition %d of topic %s: %w", offset, c.partition, c.topic, err)
			}

			if handleFailed {
				data = append(data, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameRecordsConsumedFailed, c.topic, c.partition, float64(len(records)), metric.UnitCount)...)
			}

			c.metricWriter.Write(ctx, data)
			waitStart = c.clock.Now()
		}
	}
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
