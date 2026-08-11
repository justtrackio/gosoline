package consumer

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	metricNameRebalanceCount = "RebalanceCount"
)

type PartitionManager struct {
	logger       log.Logger
	metricWriter metric.Writer
	name         string
}

func NewPartitionManager(logger log.Logger, metricWriter metric.Writer, name string) *PartitionManager {
	return &PartitionManager{
		logger:       logger,
		metricWriter: metricWriter,
		name:         name,
	}
}

func (p *PartitionManager) OnPartitionsAssigned(ctx context.Context, client *kgo.Client, assigned map[string][]int32) {
	for topic, partitions := range assigned {
		for _, partition := range partitions {
			p.logger.Debug(ctx, "starting to consume records for partition %d of topic %s", partition, topic)
		}
	}
}

func (p *PartitionManager) OnPartitionsLostOrRevoked(ctx context.Context, _ *kgo.Client, lost map[string][]int32) {
	for topic, partitions := range lost {
		for _, partition := range partitions {
			p.logger.Debug(ctx, "stopping to consume records for partition %d of topic %s", partition, topic)
		}

		dims := metric.Dimensions{kafka.DimensionClientType: kafka.DimensionConsumer, kafka.DimensionClient: p.name, kafka.DimensionTopic: topic}
		p.metricWriter.WriteOne(ctx, metric.NewMetricDatum(metricNameRebalanceCount, dims, 1.0, metric.UnitCount, metric.PriorityHigh))
	}
}
