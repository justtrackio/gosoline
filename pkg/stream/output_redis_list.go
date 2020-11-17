package stream

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
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
	logger            mon.Logger
	metricWriter      mon.MetricWriter
	client            redis.Client
	settings          *RedisListOutputSettings
	fullyQualifiedKey string
}

func NewRedisListOutput(config cfg.Config, logger mon.Logger, settings *RedisListOutputSettings) Output {
	settings.PadFromConfig(config)
	client := redis.ProvideClient(config, logger, settings.ServerName)

	defaultMetrics := getRedisListOutputDefaultMetrics(settings.AppId, settings.Key)
	mw := mon.NewMetricDaemonWriter(defaultMetrics...)

	return NewRedisListOutputWithInterfaces(logger, mw, client, settings)
}

func NewRedisListOutputWithInterfaces(logger mon.Logger, mw mon.MetricWriter, client redis.Client, settings *RedisListOutputSettings) Output {
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

func (o *redisListOutput) Write(_ context.Context, batch []WritableMessage) error {
	chunks, err := BuildChunks(batch, o.settings.BatchSize)

	if err != nil {
		o.logger.Error(err, "could not batch all messages")
	}

	for _, chunk := range chunks {
		interfaces := ByteChunkToInterfaces(chunk)
		_, err := o.client.RPush(o.fullyQualifiedKey, interfaces...)

		if err != nil {
			return err
		}
	}

	o.writeListWriteMetric(len(batch))

	return nil
}

func (o *redisListOutput) writeListWriteMetric(length int) {
	data := mon.MetricData{{
		Priority:   mon.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricNameRedisListOutputWrites,
		Dimensions: map[string]string{
			"StreamName": o.fullyQualifiedKey,
		},
		Unit:  mon.UnitCount,
		Value: float64(length),
	}}

	o.metricWriter.Write(data)
}

func getRedisListOutputDefaultMetrics(appId cfg.AppId, key string) mon.MetricData {
	fullyQualifiedKey := redis.GetFullyQualifiedKey(appId, key)

	return mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameRedisListOutputWrites,
			Dimensions: map[string]string{
				"StreamName": fullyQualifiedKey,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
	}
}
