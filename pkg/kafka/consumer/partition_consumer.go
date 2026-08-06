package consumer

import (
	"context"
	"fmt"
	"time"

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
	metricNameCommitWaitTimeouts    = "CommitWaitTimeouts"
	metricNameRecordsConsumedFailed = "RecordsConsumedFailed"
)

type PartitionConsumer struct {
	logger            log.Logger
	clock             clock.Clock
	metricWriter      metric.Writer
	name              string
	topic             string
	partition         int32
	messageHandler    KafkaMessageHandler
	kafkaClient       *kgo.Client
	commitWaitTimeout time.Duration
	assignedBatch     chan []*kgo.Record
	stop              chan struct{}
	done              chan struct{}
}

func NewPartitionConsumer(
	logger log.Logger,
	clk clock.Clock,
	metricWriter metric.Writer,
	messageHandler KafkaMessageHandler,
	kafkaClient *kgo.Client,
	commitWaitTimeout time.Duration,
	name string,
	topic string,
	partition int32,
) *PartitionConsumer {
	return &PartitionConsumer{
		logger:            logger,
		clock:             clk,
		metricWriter:      metricWriter,
		name:              name,
		topic:             topic,
		partition:         partition,
		messageHandler:    messageHandler,
		kafkaClient:       kafkaClient,
		commitWaitTimeout: commitWaitTimeout,
		assignedBatch:     make(chan []*kgo.Record),
		stop:              make(chan struct{}),
		done:              make(chan struct{}),
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
			failedCount := c.handleWithRecovery(ctx, records)
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

			if failedCount > 0 {
				data = append(data, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameRecordsConsumedFailed, c.topic, c.partition, float64(failedCount), metric.UnitCount)...)
			}

			c.metricWriter.Write(ctx, data)
			waitStart = c.clock.Now()
		}
	}
}

// handleWithRecovery hands the records over for processing and waits until all of them have been processed, so that we
// only commit their offsets afterwards. It returns the number of records which were not processed successfully.
func (c *PartitionConsumer) handleWithRecovery(ctx context.Context, records []*kgo.Record) (failed int) {
	defer func() {
		if err := coffin.ResolveRecovery(recover()); err != nil {
			c.logger.Error(ctx, "panic in message handler for partition %d of topic %s: %w", c.partition, c.topic, err)
			failed = len(records)
		}
	}()

	completion := c.messageHandler.Handle(records)

	return c.awaitCompletion(ctx, completion, len(records))
}

// awaitCompletion blocks until every record of the batch got processed. Should this take longer than the configured
// timeout, we give up waiting and let the caller commit anyway: a stalled consumer would be kicked out of the consumer
// group once it exceeds the rebalance timeout, so we rather degrade to the previous behaviour and report it loudly.
func (c *PartitionConsumer) awaitCompletion(ctx context.Context, completion BatchCompletion, recordCount int) int {
	timer := c.clock.NewTimer(c.commitWaitTimeout)
	defer timer.Stop()

	select {
	case <-completion.Done():
		return completion.FailedCount()
	case <-ctx.Done():
		// the client is going away, so committing is pointless - the records will be redelivered
		c.logger.Warn(ctx, "context cancelled while waiting for %d records of partition %d of topic %s to be processed", recordCount, c.partition, c.topic)

		return completion.FailedCount()
	case <-timer.Chan():
		c.logger.Error(ctx, "timed out after %s waiting for %d records of partition %d of topic %s to be processed, committing anyway - this might lose messages", c.commitWaitTimeout, recordCount, c.partition, c.topic)
		c.metricWriter.Write(ctx, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameCommitWaitTimeouts, c.topic, c.partition, 1.0, metric.UnitCount))

		return completion.FailedCount()
	}
}

func (c *PartitionConsumer) Stop() {
	close(c.stop)
}
