package consumer

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

type PartitionManager struct {
	logger         log.Logger
	cfn            coffin.Coffin
	consumers      map[assignment]*PartitionConsumer
	lck            *sync.Mutex
	messageHandler KafkaMessageHandler
	done           chan struct{}
}

type assignment struct {
	topic     string
	partition int32
}

func NewPartitionManager(logger log.Logger, messageHandler KafkaMessageHandler) PartitionManager {
	cfn := coffin.New()
	done := make(chan struct{})

	cfn.Go(func() error {
		<-done

		return nil
	})

	return PartitionManager{
		logger:         logger,
		cfn:            cfn,
		consumers:      make(map[assignment]*PartitionConsumer),
		lck:            &sync.Mutex{},
		messageHandler: messageHandler,
		done:           done,
	}
}

func (p *PartitionManager) OnPartitionsAssigned(ctx context.Context, client *kgo.Client, assigned map[string][]int32) {
	for topic, partitions := range assigned {
		for _, partition := range partitions {
			partitionConsumer := NewPartitionConsumer(p.logger, topic, partition, p.messageHandler, client)

			p.lck.Lock()
			p.consumers[assignment{topic, partition}] = partitionConsumer
			p.lck.Unlock()

			p.logger.Debug(ctx, "starting to consume records for partition %d of topic %s", partition, topic)

			p.cfn.Go(func() error {
				err := partitionConsumer.Consume(ctx)
				if err != nil {
					return fmt.Errorf("failed to consume records for partition %d of topic %s: %w", partition, topic, err)
				}

				return nil
			})
		}
	}
}

func (p *PartitionManager) OnPartitionsLostOrRevoked(ctx context.Context, _ *kgo.Client, lost map[string][]int32) {
	var wg sync.WaitGroup
	defer wg.Wait()

	for topic, partitions := range lost {
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
			p.logger.Debug(ctx, "waiting for work to finish for lost/revoked partition %d of topic %s", partition, topic)

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

func (p *PartitionManager) Handle(topic string, partition int32, records []*kgo.Record) {
	p.lck.Lock()
	defer p.lck.Unlock()

	p.consumers[assignment{topic, partition}].assignedBatch <- records
}

func (p *PartitionManager) HandleWithoutCommit(records []*kgo.Record) {
	p.messageHandler.Handle(records)
}

func (p *PartitionManager) Stop(ctx context.Context) {
	p.lck.Lock()
	for _, consumer := range p.consumers {
		consumer.Stop()
	}
	p.consumers = map[assignment]*PartitionConsumer{}
	p.lck.Unlock()

	close(p.done)

	if err := p.cfn.Wait(); err != nil {
		p.logger.Error(ctx, "failed to stop partition consumers: %w", err)
	}

	p.messageHandler.Stop()
}
