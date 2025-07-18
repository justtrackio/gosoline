package stream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/redis"
)

const (
	metricNameRedisListInputLength = "StreamRedisListInputLength"
	metricNameRedisListInputReads  = "StreamRedisListInputReads"
)

type RedisListInputSettings struct {
	cfg.AppId
	ServerName         string
	Key                string
	WaitTime           time.Duration
	HealthcheckTimeout time.Duration
}

type redisListInput struct {
	logger           log.Logger
	mw               metric.Writer
	client           redis.Client
	settings         *RedisListInputSettings
	healthCheckTimer clock.HealthCheckTimer

	channel           chan *Message
	stopped           bool
	fullyQualifiedKey string
}

func NewRedisListInput(ctx context.Context, config cfg.Config, logger log.Logger, settings *RedisListInputSettings) (Input, error) {
	settings.PadFromConfig(config)

	client, err := redis.ProvideClient(ctx, config, logger, settings.ServerName)
	if err != nil {
		return nil, fmt.Errorf("can not create redis client: %w", err)
	}

	defaultMetrics := getRedisListInputDefaultMetrics(settings.AppId, settings.Key)
	mw := metric.NewWriter(defaultMetrics...)

	healthCheckTimer, err := clock.NewHealthCheckTimer(settings.HealthcheckTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create healthcheck timer: %w", err)
	}

	return NewRedisListInputWithInterfaces(logger, client, mw, settings, healthCheckTimer), nil
}

func NewRedisListInputWithInterfaces(
	logger log.Logger,
	client redis.Client,
	mw metric.Writer,
	settings *RedisListInputSettings,
	healthCheckTimer clock.HealthCheckTimer,
) Input {
	fullyQualifiedKey := redis.GetFullyQualifiedKey(settings.AppId, settings.Key)

	return &redisListInput{
		logger:            logger,
		client:            client,
		settings:          settings,
		healthCheckTimer:  healthCheckTimer,
		mw:                mw,
		channel:           make(chan *Message),
		fullyQualifiedKey: fullyQualifiedKey,
	}
}

func (i *redisListInput) Data() <-chan *Message {
	return i.channel
}

func (i *redisListInput) Run(ctx context.Context) error {
	defer close(i.channel)

	if i.settings.WaitTime == 0 {
		return errors.New("wait time should be bigger than 0")
	}

	go i.runMetricLoop(ctx)

	for {
		if i.stopped {
			return nil
		}

		i.healthCheckTimer.MarkHealthy()

		rawMessage, err := i.client.BLPop(ctx, i.settings.WaitTime, i.fullyQualifiedKey)

		if err != nil && err.Error() != redis.Nil.Error() {
			i.logger.Error("could not BLPop from redis: %w", err)
			i.stopped = true

			return err
		}

		if len(rawMessage) == 0 {
			continue
		}

		msg := Message{}
		err = json.Unmarshal([]byte(rawMessage[1]), &msg)
		if err != nil {
			i.logger.Error("could not unmarshal message: %w", err)

			continue
		}

		i.channel <- &msg
		i.writeListReadMetric()
	}
}

func (i *redisListInput) Stop() {
	i.stopped = true
}

func (i *redisListInput) IsHealthy() bool {
	return i.healthCheckTimer.IsHealthy()
}

func (i *redisListInput) runMetricLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)

	for {
		i.writeListLengthMetric(ctx)
		<-ticker.C
	}
}

func (i *redisListInput) writeListLengthMetric(ctx context.Context) {
	llen, err := i.client.LLen(ctx, i.fullyQualifiedKey)
	if err != nil {
		i.logger.Error("can not publish stream list metric data: %w", err)

		return
	}

	data := metric.Data{{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameRedisListInputLength,
		Dimensions: map[string]string{
			"StreamName": i.fullyQualifiedKey,
		},
		Unit:  metric.UnitCountAverage,
		Value: float64(llen),
	}}

	i.mw.Write(data)
}

func (i *redisListInput) writeListReadMetric() {
	data := metric.Data{{
		MetricName: metricNameRedisListInputReads,
		Dimensions: map[string]string{
			"StreamName": i.fullyQualifiedKey,
		},
		Value: 1.0,
	}}

	i.mw.Write(data)
}

func getRedisListInputDefaultMetrics(appId cfg.AppId, key string) metric.Data {
	fullyQualifiedKey := redis.GetFullyQualifiedKey(appId, key)

	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameRedisListInputReads,
			Dimensions: map[string]string{
				"StreamName": fullyQualifiedKey,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
