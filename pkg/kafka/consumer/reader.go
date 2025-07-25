package consumer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

//go:generate go run github.com/vektra/mockery/v2 --name Reader
type Reader interface {
	AllowRebalance()
	CloseAllowingRebalance()
	PollRecords(ctx context.Context, maxPollRecords int) kgo.Fetches
}

func NewReader(ctx context.Context, config cfg.Config, logger log.Logger, settingsKey string, options ...kgo.Opt) (Reader, error) {
	settings, err := ParseSettings(config, settingsKey)
	if err != nil {
		return nil, fmt.Errorf("kafka: failed to parse consumer settings: %w", err)
	}

	// todo: add TLS/Dialer configuration

	opts := []kgo.Opt{
		kgo.Balancers(settings.GetBalancer()),
		kgo.BlockRebalanceOnPoll(),
		kgo.ConsumeResetOffset(settings.GetStartOffset()),
		kgo.ConsumeStartOffset(settings.GetStartOffset()),
		kgo.ConsumerGroup(settings.FQGroupID),
		kgo.ConsumeTopics(settings.FQTopic),
		kgo.DisableAutoCommit(),
		kgo.SeedBrokers(settings.Connection().Bootstrap...),
		kgo.WithContext(ctx),
		kgo.WithLogger(logging.NewKafkaLogger(logger)),
	}

	opts = append(opts, options...)

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("can not create franz-go client: %w", err)
	}

	return client, nil
}
