package consumer

import (
	"context"
	"sync"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

type PartitionManager struct {
	logger         log.Logger
	consumers      map[assignment]*PartitionConsumer
	messageHandler KafkaMessageHandler
}

type assignment struct {
	topic     string
	partition int32
}

func NewPartitionManager(logger log.Logger, messageHandler KafkaMessageHandler) *PartitionManager {
	return &PartitionManager{
		logger:         logger,
		consumers:      make(map[assignment]*PartitionConsumer),
		messageHandler: messageHandler,
	}
}

func (p PartitionManager) OnPartitionsAssigned(ctx context.Context, client *kgo.Client, assigned map[string][]int32) {
	for topic, partitions := range assigned {
		for _, partition := range partitions {
			partitionConsumer := NewPartitionConsumer(p.logger, topic, partition, p.messageHandler, client)
			p.consumers[assignment{topic, partition}] = partitionConsumer

			p.logger.WithContext(ctx).Debug("starting to consume records for partition %d of topic %s", partition, topic)
			go partitionConsumer.Consume(ctx)
		}
	}
}

func (p PartitionManager) OnPartitionsLostOrRevoked(ctx context.Context, _ *kgo.Client, lost map[string][]int32) {
	var wg sync.WaitGroup
	defer wg.Wait()

	for topic, partitions := range lost {
		for _, partition := range partitions {
			assignment := assignment{topic, partition}
			partitionConsumer := p.consumers[assignment]

			delete(p.consumers, assignment)

			partitionConsumer.Stop()
			p.logger.WithContext(ctx).Debug("waiting for work to finish for lost/revoked partition %d of topic %s", partition, topic)

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

func (p PartitionManager) AssignRecords(topic string, partition int32, records []*kgo.Record) {
	p.consumers[assignment{topic, partition}].assignedBatch <- records
}
