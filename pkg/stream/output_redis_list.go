package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/metric"
	"github.com/applike/gosoline/pkg/redis"
	"time"
)

const (
	metricNameRedisListOutputWrites = "StreamRedisListOutputWrites"
)

type RedisListOutputSettings struct {
	cfg.AppId
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

func NewRedisListOutput(config cfg.Config, logger log.Logger, settings *RedisListOutputSettings) (Output, error) {
	settings.PadFromConfig(config)

	client, err := redis.ProvideClient(config, logger, settings.ServerName)
	if err != nil {
		return nil, fmt.Errorf("can not create redis client: %w", err)
	}

	defaultMetrics := getRedisListOutputDefaultMetrics(settings.AppId, settings.Key)
	mw := metric.NewDaemonWriter(defaultMetrics...)

	return NewRedisListOutputWithInterfaces(logger, mw, client, settings), nil
}

func NewRedisListOutputWithInterfaces(logger log.Logger, mw metric.Writer, client redis.Client, settings *RedisListOutputSettings) Output {
	fullyQualifiedKey := redis.GetFullyQualifiedKey(settings.AppId, settings.Key)

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
		o.logger.Error("could not batch all messages: %w", err)
	}

	for _, chunk := range chunks {
		interfaces := ByteChunkToInterfaces(chunk)
		_, err := o.client.RPush(ctx, o.fullyQualifiedKey, interfaces...)

		if err != nil {
			return err
		}
	}

	o.writeListWriteMetric(len(batch))

	return nil
}

func (o *redisListOutput) writeListWriteMetric(length int) {
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

	o.metricWriter.Write(data)
}

func getRedisListOutputDefaultMetrics(appId cfg.AppId, key string) metric.Data {
	fullyQualifiedKey := redis.GetFullyQualifiedKey(appId, key)

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
	}
}
