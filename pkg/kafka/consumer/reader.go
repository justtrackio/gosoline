package consumer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/twmb/franz-go/pkg/kgo"
)

//go:generate go run github.com/vektra/mockery/v2 --name Reader
type Reader interface {
	AllowRebalance()
	CloseAllowingRebalance()
	PollRecords(ctx context.Context, maxPollRecords int) kgo.Fetches
}

func NewReader(ctx context.Context, config cfg.Config, logger log.Logger, settings Settings, partitionManager PartitionManager, isReadOnly bool) (Reader, error) {
	if err := settings.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("failed to pad app id from config: %w", err)
	}

	topicName, err := kafka.BuildFullTopicName(config, settings.AppId, settings.TopicId)
	if err != nil {
		return nil, fmt.Errorf("failed to build full kafka topic name: %w", err)
	}

	opts := []kgo.Opt{
		kgo.ConsumeResetOffset(settings.GetStartOffset()),
		kgo.ConsumeStartOffset(settings.GetStartOffset()),
		kgo.ConsumeTopics(topicName),
		kgo.HeartbeatInterval(settings.HeartbeatInterval),
		kgo.RebalanceTimeout(settings.RebalanceTimeout),
		kgo.SessionTimeout(settings.SessionTimeout),
		kgo.WithContext(ctx),
		kgo.WithLogger(logging.NewKafkaLogger(ctx, logger)),
	}

	if !isReadOnly {
		consumerGroupId, err := kafka.BuildFullConsumerGroupId(config, settings.GroupId)
		if err != nil {
			return nil, fmt.Errorf("failed to build full kafka consumer group id: %w", err)
		}

		opts = append(opts, []kgo.Opt{
			kgo.Balancers(settings.GetBalancers()...),
			kgo.BlockRebalanceOnPoll(),
			kgo.ConsumerGroup(consumerGroupId),
			kgo.DisableAutoCommit(),
			kgo.OnPartitionsAssigned(partitionManager.OnPartitionsAssigned),
			kgo.OnPartitionsRevoked(partitionManager.OnPartitionsLostOrRevoked),
			kgo.OnPartitionsLost(partitionManager.OnPartitionsLostOrRevoked),
		}...)
	}

	connOpts, err := connection.BuildConnectionOptions(config, settings.Connection)
	if err != nil {
		return nil, fmt.Errorf("failed to build connection options: %w", err)
	}
	opts = append(opts, connOpts...)

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("can not create franz-go client: %w", err)
	}

	if err = reslife.AddLifeCycleer(ctx, kafka.NewLifecycleManager(settings.Connection, topicName)); err != nil {
		return nil, fmt.Errorf("failed to add kafka lifecycle manager: %w", err)
	}

	return client, nil
}
