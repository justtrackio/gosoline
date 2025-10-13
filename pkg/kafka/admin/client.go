package admin

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

//go:generate go run github.com/vektra/mockery/v2 --name Client
type Client interface {
	CreateTopic(ctx context.Context, partitions int32, replicationFactor int16, configs map[string]*string, topic string) (kadm.CreateTopicResponse, error)
	ListEndOffsets(ctx context.Context, topics ...string) (kadm.ListedOffsets, error)
	ListTopics(ctx context.Context, topics ...string) (kadm.TopicDetails, error)
}

func NewClient(ctx context.Context, logger log.Logger, brokers []string) (Client, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.WithContext(ctx),
		kgo.WithLogger(logging.NewKafkaLogger(ctx, logger)),
	}

	return kadm.NewOptClient(opts...)
}
