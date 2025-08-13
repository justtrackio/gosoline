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

func NewReader(ctx context.Context, config cfg.Config, logger log.Logger, settings Settings, additionalOptions ...kgo.Opt) (Reader, error) {
	appId, err := cfg.GetAppIdFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get app id from config: %w", err)
	}

	consumerGroupId, err := kafka.BuildFullConsumerGroupId(config, appId, settings.GroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to build full kafka consumer group id: %w", err)
	}

	topicName, err := kafka.BuildFullTopicName(config, appId, settings.TopicId)
	if err != nil {
		return nil, fmt.Errorf("failed to build full kafka topic name: %w", err)
	}

	opts := []kgo.Opt{
		kgo.Balancers(settings.GetBalancer()),
		kgo.BlockRebalanceOnPoll(),
		kgo.ConsumeResetOffset(settings.GetStartOffset()),
		kgo.ConsumeStartOffset(settings.GetStartOffset()),
		kgo.ConsumerGroup(consumerGroupId),
		kgo.ConsumeTopics(topicName),
		kgo.DisableAutoCommit(),
		kgo.HeartbeatInterval(settings.HeartbeatInterval),
		kgo.RebalanceTimeout(settings.RebalanceTimeout),
		kgo.SessionTimeout(settings.SessionTimeout),
		kgo.WithContext(ctx),
		kgo.WithLogger(logging.NewKafkaLogger(logger)),
	}

	connOpts, err := connection.BuildConnectionOptions(config, settings.Connection)
	if err != nil {
		return nil, fmt.Errorf("failed to build connection options: %w", err)
	}
	opts = append(opts, connOpts...)

	opts = append(opts, additionalOptions...)

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("can not create franz-go client: %w", err)
	}

	if err = reslife.AddLifeCycleer(ctx, kafka.NewLifecycleManager(settings.Connection, topicName)); err != nil {
		return nil, fmt.Errorf("failed to add kafka lifecycle manager: %w", err)
	}

	return client, nil
}
