package consumer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	metricNameProcessDuration       = "ProcessDuration"
	metricNameWaitDuration          = "WaitDuration"
	metricNameSleepDuration         = "SleepDuration"
	metricNameCommitDuration        = "CommitDuration"
	metricNameCommitFailures        = "CommitFailures"
	metricNameRecordsConsumedFailed = "RecordsConsumedFailed"
)

//go:generate go run github.com/vektra/mockery/v2 --name partitionClient --structname PartitionClient --exported
type partitionClient interface {
	CommitRecords(ctx context.Context, records ...*kgo.Record) error
	PauseFetchPartitions(topicPartitions map[string][]int32) map[string][]int32
	ResumeFetchPartitions(topicPartitions map[string][]int32)
}

type PartitionConsumer struct {
	logger         log.Logger
	clock          clock.Clock
	metricWriter   metric.Writer
	name           string
	topic          string
	partition      int32
	messageHandler KafkaMessageHandler
	kafkaClient    partitionClient
	assignedBatch  chan []*kgo.Record
	stop           chan struct{}
	drain          chan struct{}
	done           chan struct{}
	stopOnce       sync.Once
	drainOnce      sync.Once
	delayConsume   func(ctx context.Context, stopped <-chan struct{}, records []*kgo.Record) bool
	pauseFetch     bool
	paused         atomic.Bool
}

func newPartitionConsumer(
	logger log.Logger,
	clk clock.Clock,
	metricWriter metric.Writer,
	messageHandler KafkaMessageHandler,
	kafkaClient partitionClient,
	name, topic string,
	partition int32,
	pauseFetch bool,
	consumeDelay func(ctx context.Context, stopped <-chan struct{}, records []*kgo.Record) bool,
) *PartitionConsumer {
	batchBufferSize := 0
	if pauseFetch {
		batchBufferSize = 1
	}

	return &PartitionConsumer{
		logger:         logger,
		clock:          clk,
		metricWriter:   metricWriter,
		name:           name,
		topic:          topic,
		partition:      partition,
		messageHandler: messageHandler,
		kafkaClient:    kafkaClient,
		assignedBatch:  make(chan []*kgo.Record, batchBufferSize),
		stop:           make(chan struct{}),
		drain:          make(chan struct{}),
		done:           make(chan struct{}),
		delayConsume:   consumeDelay,
		pauseFetch:     pauseFetch,
	}
}

func (c *PartitionConsumer) Consume(ctx context.Context) error {
	defer c.logger.Debug(ctx, "done consuming partition %d of topic %s", c.partition, c.topic)
	defer c.resumeFetch()
	defer close(c.done)

	waitStart := c.clock.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stop:
			return nil
		case <-c.drain:
			return c.consumePending(ctx, waitStart)
		case records := <-c.assignedBatch:
			if len(records) == 0 {
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-c.stop:
				return nil
			default:
			}

			keepConsuming, err := c.consumeBatch(ctx, records, waitStart)
			if err != nil {
				return err
			}
			if !keepConsuming {
				return nil
			}

			waitStart = c.clock.Now()
		}
	}
}

func (c *PartitionConsumer) consumePending(ctx context.Context, waitStart time.Time) error {
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

			keepConsuming, err := c.consumeBatch(ctx, records, waitStart)
			if err != nil {
				return err
			}
			if !keepConsuming {
				return nil
			}

			waitStart = c.clock.Now()
		default:
			return nil
		}
	}
}

func (c *PartitionConsumer) Assign(ctx context.Context, records []*kgo.Record) {
	if len(records) == 0 {
		return
	}

	select {
	case <-ctx.Done():
	case <-c.stop:
	case <-c.drain:
	case <-c.done:
	default:
		if c.pauseFetch {
			c.kafkaClient.PauseFetchPartitions(c.topicPartition())
			c.paused.Store(true)
		}

		select {
		case c.assignedBatch <- records:
			return
		case <-ctx.Done():
		case <-c.stop:
		case <-c.drain:
		case <-c.done:
		}
	}

	c.resumeFetch()
}

func (c *PartitionConsumer) consumeBatch(ctx context.Context, records []*kgo.Record, waitStart time.Time) (bool, error) {
	defer c.resumeFetch()

	waitMs := float64(c.clock.Since(waitStart).Milliseconds())
	sleepStart := c.clock.Now()

	if !c.delayConsume(ctx, c.stop, records) {
		return false, nil
	}

	sleepMs := float64(c.clock.Since(sleepStart).Milliseconds())
	processStart := c.clock.Now()
	handleFailed := c.handleWithRecovery(ctx, records)
	processMs := float64(c.clock.Since(processStart).Milliseconds())

	commitStart := c.clock.Now()
	err := c.kafkaClient.CommitRecords(ctx, records...)
	commitMs := float64(c.clock.Since(commitStart).Milliseconds())

	var data metric.Data
	data = append(data, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameWaitDuration, c.topic, c.partition, waitMs, metric.UnitMillisecondsAverage)...)
	data = append(data, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameSleepDuration, c.topic, c.partition, sleepMs, metric.UnitMillisecondsAverage)...)
	data = append(data, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameProcessDuration, c.topic, c.partition, processMs, metric.UnitMillisecondsAverage)...)
	data = append(data, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameCommitDuration, c.topic, c.partition, commitMs, metric.UnitMillisecondsAverage)...)

	if err != nil {
		data = append(data, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameCommitFailures, c.topic, c.partition, 1.0, metric.UnitCount)...)

		c.metricWriter.Write(ctx, data)

		// The partition was revoked or lost while its offset commit was in flight. Kafka rejects
		// commits from the old group generation; the next assignment creates a new worker, and
		// records whose offsets were not committed may be delivered again.
		if c.isStopped() && isRebalanceCommitError(err) {
			c.logger.Warn(ctx, "ignoring commit failure for lost or revoked partition %d of topic %s: %s", c.partition, c.topic, err.Error())

			return false, nil
		}

		offset := records[len(records)-1].Offset + 1

		return false, fmt.Errorf("failed to commit offset %d for partition %d of topic %s: %w", offset, c.partition, c.topic, err)
	}

	if handleFailed {
		data = append(data, kafka.MetricPair(kafka.DimensionConsumer, c.name, metricNameRecordsConsumedFailed, c.topic, c.partition, float64(len(records)), metric.UnitCount)...)
	}

	c.metricWriter.Write(ctx, data)

	return true, nil
}

func (c *PartitionConsumer) isStopped() bool {
	select {
	case <-c.stop:
		return true
	default:
		return false
	}
}

func (c *PartitionConsumer) resumeFetch() {
	if c.pauseFetch && c.paused.Swap(false) {
		c.kafkaClient.ResumeFetchPartitions(c.topicPartition())
	}
}

func (c *PartitionConsumer) topicPartition() map[string][]int32 {
	return map[string][]int32{c.topic: {c.partition}}
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
	c.stopOnce.Do(func() {
		close(c.stop)
	})
}

func (c *PartitionConsumer) Drain() {
	c.drainOnce.Do(func() {
		close(c.drain)
	})
}

func isRebalanceCommitError(err error) bool {
	return errors.Is(err, kerr.IllegalGeneration) ||
		errors.Is(err, kerr.UnknownMemberID) ||
		errors.Is(err, kerr.RebalanceInProgress)
}
