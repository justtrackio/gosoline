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
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	metricNameRebalanceCount = "RebalanceCount"
)

type PartitionManager struct {
	logger         log.Logger
	clock          clock.Clock
	metricWriter   metric.Writer
	name           string
	cfn            coffin.Coffin
	consumers      map[assignment]*PartitionConsumer
	lck            sync.RWMutex
	messageHandler KafkaMessageHandler
	done           chan struct{}
	errors         chan error
	stopping       atomic.Bool
	consumeDelay   time.Duration
	healthCheck    clock.HealthCheckTimer
	healthTimeout  time.Duration
}

type assignment struct {
	topic     string
	partition int32
}

func newPartitionManager(
	logger log.Logger,
	clk clock.Clock,
	metricWriter metric.Writer,
	messageHandler KafkaMessageHandler,
	name string,
	consumeDelay time.Duration,
	healthCheck clock.HealthCheckTimer,
	healthTimeout time.Duration,
) *PartitionManager {
	cfn := coffin.New()
	done := make(chan struct{})

	cfn.Go(func() error {
		<-done

		return nil
	})

	return &PartitionManager{
		logger:         logger,
		clock:          clk,
		metricWriter:   metricWriter,
		name:           name,
		cfn:            cfn,
		consumers:      make(map[assignment]*PartitionConsumer),
		messageHandler: messageHandler,
		done:           done,
		errors:         make(chan error, 1),
		consumeDelay:   consumeDelay,
		healthCheck:    healthCheck,
		healthTimeout:  healthTimeout,
	}
}

func (p *PartitionManager) OnPartitionsAssigned(ctx context.Context, client *kgo.Client, assigned map[string][]int32) {
	if p.stopping.Load() {
		p.logger.Info(ctx, "ignoring partition assignment while partition manager is stopping")

		return
	}

	for topic, partitions := range assigned {
		for _, partition := range partitions {
			p.lck.Lock()

			if p.stopping.Load() {
				p.lck.Unlock()
				p.logger.Info(ctx, "ignoring partition assignment for partition %d of topic %s while partition manager is stopping", partition, topic)

				continue
			}

			partitionConsumer := newPartitionConsumer(p.logger, p.clock, p.metricWriter, p.messageHandler, client, p.name, topic, partition, p.consumeDelay > 0, p.delayConsume)

			p.consumers[assignment{topic, partition}] = partitionConsumer
			p.lck.Unlock()

			p.logger.Info(ctx, "starting to consume records for partition %d of topic %s", partition, topic)

			p.cfn.Go(func() error {
				return p.consumePartition(ctx, partitionConsumer)
			})
		}
	}
}

func (p *PartitionManager) consumePartition(ctx context.Context, partitionConsumer *PartitionConsumer) error {
	err := partitionConsumer.Consume(ctx)
	if err == nil {
		return nil
	}
	if p.stopping.Load() && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		return nil
	}

	err = fmt.Errorf("failed to consume records for partition %d of topic %s: %w", partitionConsumer.partition, partitionConsumer.topic, err)
	select {
	case p.errors <- err:
	default:
	}

	return err
}

func (p *PartitionManager) OnPartitionsLostOrRevoked(ctx context.Context, _ *kgo.Client, lost map[string][]int32) {
	var wg sync.WaitGroup
	defer wg.Wait()

	for topic, partitions := range lost {
		dims := metric.Dimensions{kafka.DimensionClientType: kafka.DimensionConsumer, kafka.DimensionClient: p.name, kafka.DimensionTopic: topic}
		p.metricWriter.WriteOne(ctx, metric.NewMetricDatum(metricNameRebalanceCount, dims, 1.0, metric.UnitCount, metric.PriorityHigh))

		for _, partition := range partitions {
			assignment := assignment{topic, partition}

			p.lck.Lock()
			partitionConsumer, ok := p.consumers[assignment]
			delete(p.consumers, assignment)
			p.lck.Unlock()

			if !ok {
				continue
			}

			partitionConsumer.Stop()
			p.logger.Info(ctx, "waiting for work to finish for lost/revoked partition %d of topic %s", partition, topic)

			// as long as we are here we are blocking a rebalance.
			// we should take advantage of that and wait until all consumers for the revoked partitions are done.
			// otherwise we would allow a rebalance of the revoked partitions while we are still processing potentially uncommitted messages,
			// which would then be processed again by another consumer.
			wg.Add(1)
			go func() {
				<-partitionConsumer.done
				wg.Done()
			}()
		}
	}
}

func (p *PartitionManager) Handle(ctx context.Context, topic string, partition int32, records []*kgo.Record) {
	p.lck.RLock()
	consumer, ok := p.consumers[assignment{topic, partition}]
	p.lck.RUnlock()

	if !ok {
		// at the time Handle is called, we are blocking a rebalance and OnPartitionsLostOrRevoked is only called once a rebalance is allowed again, so this should never happen
		p.logger.Error(ctx, "no consumer found for partition %d of topic %s", partition, topic)

		return
	}

	consumer.Assign(ctx, records)
}

func (p *PartitionManager) handleWithoutCommit(ctx context.Context, stopped <-chan struct{}, records []*kgo.Record) {
	if p.delayConsumeReadOnly(ctx, stopped, records) {
		p.messageHandler.Handle(records)
	}
}

func (p *PartitionManager) delayConsume(ctx context.Context, stopped <-chan struct{}, records []*kgo.Record) bool {
	return p.waitConsumeDelay(ctx, stopped, records, false)
}

func (p *PartitionManager) delayConsumeReadOnly(ctx context.Context, stopped <-chan struct{}, records []*kgo.Record) bool {
	return p.waitConsumeDelay(ctx, stopped, records, true)
}

func (p *PartitionManager) waitConsumeDelay(ctx context.Context, stopped <-chan struct{}, records []*kgo.Record, keepHealthy bool) bool {
	if p.consumeDelay == 0 {
		return true
	}

	var latestTimestamp time.Time
	for _, record := range records {
		if record != nil && record.Timestamp.After(latestTimestamp) {
			latestTimestamp = record.Timestamp
		}
	}

	durationToSleep := p.consumeDelay - p.clock.Since(latestTimestamp)
	if durationToSleep <= 0 {
		return true
	}

	timer := p.clock.NewTimer(durationToSleep)
	defer timer.Stop()

	if keepHealthy && p.healthCheck != nil {
		p.healthCheck.MarkHealthy()
	}

	var healthTicker clock.Ticker
	var healthTickerChan <-chan time.Time

	if keepHealthy && p.healthCheck != nil && p.healthTimeout > 0 {
		healthInterval := p.healthTimeout / 2

		if healthInterval <= 0 {
			healthInterval = p.healthTimeout
		}

		healthTicker = p.clock.NewTicker(healthInterval)
		healthTickerChan = healthTicker.Chan()
		defer healthTicker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			return false
		case <-stopped:
			return false
		case <-timer.Chan():
			return true
		case <-healthTickerChan:
			p.healthCheck.MarkHealthy()
		}
	}
}

func (p *PartitionManager) Stop(ctx context.Context) {
	p.stopping.Store(true)

	p.lck.Lock()
	partitionCount := len(p.consumers)

	p.logger.Info(ctx, "draining %d Kafka partitions for consumer %s", partitionCount, p.name)

	for assignment, consumer := range p.consumers {
		consumer.Drain()
		delete(p.consumers, assignment)
	}
	p.lck.Unlock()

	close(p.done)

	// Partition errors are escalated through p.errors and logged by the module owner.
	// Waiting here must not report the coffin's copy of the same error again.
	<-p.cfn.Dead()

	if err := ctx.Err(); err != nil {
		p.logger.Warn(ctx, "Kafka partition drain for consumer %s ended after the grace period: %s", p.name, err)
	} else {
		p.logger.Info(ctx, "successfully drained %d Kafka partitions for consumer %s", partitionCount, p.name)
	}

	p.messageHandler.Stop()
}
