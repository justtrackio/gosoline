package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/redis"
)

const (
	metricNameRedisListOutputWrites = "StreamRedisListOutputWrites"
)

type RedisListOutputSettings struct {
	cfg.AppIdentity
	ServerName string
	Key        string
	BatchSize  int
}

type redisListOutput struct {
	logger            log.Logger
	metricWriter      metric.Writer
	client            redis.Client
	settings          *RedisListOutputSettings
	fullyQualifiedKey string
}

func NewRedisListOutput(ctx context.Context, config cfg.Config, logger log.Logger, settings *RedisListOutputSettings) (Output, error) {
	err := settings.PadFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("can not pad settings from config: %w", err)
	}

	var client redis.Client
	client, err = redis.ProvideClient(ctx, config, logger, settings.ServerName)
	if err != nil {
		return nil, fmt.Errorf("can not create redis client: %w", err)
	}

	fullyQualifiedKey, err := redis.BuildFullyQualifiedKey(config, settings.AppIdentity, settings.Key)
	if err != nil {
		return nil, fmt.Errorf("can not build fully qualified key: %w", err)
	}

	defaultMetrics, err := getRedisListOutputDefaultMetrics(config, settings.AppIdentity, settings.Key)
	if err != nil {
		return nil, fmt.Errorf("can not build default metrics: %w", err)
	}
	mw := metric.NewWriter(defaultMetrics...)

	return NewRedisListOutputWithInterfaces(config, logger, mw, client, settings, fullyQualifiedKey), nil
}

func NewRedisListOutputWithInterfaces(config cfg.Config, logger log.Logger, mw metric.Writer, client redis.Client, settings *RedisListOutputSettings, fullyQualifiedKey string) Output {
	return &redisListOutput{
		logger:            logger,
		metricWriter:      mw,
		client:            client,
		settings:          settings,
		fullyQualifiedKey: fullyQualifiedKey,
	}
}

func (o *redisListOutput) WriteOne(ctx context.Context, record WritableMessage) error {
	return o.Write(ctx, []WritableMessage{record})
}

func (o *redisListOutput) Write(ctx context.Context, batch []WritableMessage) error {
	chunks, err := BuildChunks(batch, o.settings.BatchSize)
	if err != nil {
		o.logger.Error(ctx, "could not batch all messages: %w", err)
	}

	for _, chunk := range chunks {
		interfaces := ByteChunkToInterfaces(chunk)
		_, err := o.client.RPush(ctx, o.fullyQualifiedKey, interfaces...)
		if err != nil {
			return err
		}
	}

	o.writeListWriteMetric(ctx, len(batch))

	return nil
}

func (o *redisListOutput) writeListWriteMetric(ctx context.Context, length int) {
	data := metric.Data{{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricNameRedisListOutputWrites,
		Dimensions: map[string]string{
			"StreamName": o.fullyQualifiedKey,
		},
		Unit:  metric.UnitCount,
		Value: float64(length),
	}}

	o.metricWriter.Write(ctx, data)
}

func getRedisListOutputDefaultMetrics(config cfg.Config, appIdentity cfg.AppIdentity, key string) (metric.Data, error) {
	fullyQualifiedKey, err := redis.BuildFullyQualifiedKey(config, appIdentity, key)
	if err != nil {
		return nil, fmt.Errorf("can not build fully qualified key: %w", err)
	}

	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameRedisListOutputWrites,
			Dimensions: map[string]string{
				"StreamName": fullyQualifiedKey,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}, nil
}
