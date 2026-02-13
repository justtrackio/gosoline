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
	ServerName string
	Key        string
	BatchSize  int
}

type redisListOutput struct {
	logger       log.Logger
	metricWriter metric.Writer
	client       redis.Client
	settings     *RedisListOutputSettings
}

func NewRedisListOutput(ctx context.Context, config cfg.Config, logger log.Logger, settings *RedisListOutputSettings) (Output, error) {
	var err error
	var client redis.Client

	if client, err = redis.ProvideClient(ctx, config, logger, settings.ServerName); err != nil {
		return nil, fmt.Errorf("can not create redis client: %w", err)
	}

	defaultMetrics := getRedisListOutputDefaultMetrics(settings)
	mw := metric.NewWriter(defaultMetrics...)

	return NewRedisListOutputWithInterfaces(config, logger, mw, client, settings), nil
}

func NewRedisListOutputWithInterfaces(config cfg.Config, logger log.Logger, mw metric.Writer, client redis.Client, settings *RedisListOutputSettings) Output {
	return &redisListOutput{
		logger:       logger,
		metricWriter: mw,
		client:       client,
		settings:     settings,
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
		_, err := o.client.RPush(ctx, o.settings.Key, interfaces...)
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
			"StreamName": fmt.Sprintf("%s-%s", o.settings.ServerName, o.settings.Key),
		},
		Unit:  metric.UnitCount,
		Value: float64(length),
	}}

	o.metricWriter.Write(ctx, data)
}

func getRedisListOutputDefaultMetrics(settings *RedisListOutputSettings) metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameRedisListOutputWrites,
			Dimensions: map[string]string{
				"StreamName": fmt.Sprintf("%s-%s", settings.ServerName, settings.Key),
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
