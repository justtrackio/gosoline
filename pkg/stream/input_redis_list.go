package stream

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/conc"
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
	ServerName         string
	Key                string
	WaitTime           time.Duration
	HealthcheckTimeout time.Duration
	RunnerCount        int
}

type redisListInput struct {
	logger           log.Logger
	mw               metric.Writer
	client           redis.Client
	settings         *RedisListInputSettings
	healthCheckTimer clock.HealthCheckTimer

	stopped conc.SignalOnce
}

var _ Input = &redisListInput{}

func NewRedisListInput(ctx context.Context, config cfg.Config, logger log.Logger, settings *RedisListInputSettings) (Input, error) {
	var err error
	var client redis.Client

	if client, err = redis.ProvideClient(ctx, config, logger, settings.ServerName); err != nil {
		return nil, fmt.Errorf("can not create redis client: %w", err)
	}

	defaultMetrics := getRedisListInputDefaultMetrics(settings)
	mw := metric.NewWriter(defaultMetrics...)

	healthCheckTimer, err := clock.NewHealthCheckTimer(settings.HealthcheckTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create healthcheck timer: %w", err)
	}

	return NewRedisListInputWithInterfaces(config, logger, client, mw, settings, healthCheckTimer), nil
}

func NewRedisListInputWithInterfaces(
	config cfg.Config,
	logger log.Logger,
	client redis.Client,
	mw metric.Writer,
	settings *RedisListInputSettings,
	healthCheckTimer clock.HealthCheckTimer,
) Input {
	return &redisListInput{
		logger:           logger,
		client:           client,
		settings:         settings,
		healthCheckTimer: healthCheckTimer,
		mw:               mw,
		stopped:          conc.NewSignalOnce(),
	}
}

func (i *redisListInput) Run(ctx context.Context, process InputProcess) error {
	if i.settings.WaitTime == 0 {
		return errors.New("wait time should be bigger than 0")
	}

	runnerCount := i.settings.RunnerCount
	if runnerCount <= 0 {
		runnerCount = 1
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-i.stopped.Channel():
			cancel()
		case <-runCtx.Done():
		}
	}()

	messages := make(chan *Message)
	var workers sync.WaitGroup
	for range runnerCount {
		workers.Go(func() {
			for msg := range messages {
				process(ctx, msg)
			}
		})
	}
	defer func() {
		close(messages)
		workers.Wait()
	}()

	go i.runMetricLoop(runCtx)

	return i.runMessageLoop(ctx, runCtx, messages)
}

func (i *redisListInput) runMessageLoop(ctx context.Context, runCtx context.Context, messages chan<- *Message) error {
	for {
		if runCtx.Err() != nil {
			return nil
		}

		i.healthCheckTimer.MarkHealthy()

		rawMessage, err := i.client.BLPop(ctx, i.settings.WaitTime, i.settings.Key)

		if err != nil && err.Error() != redis.Nil.Error() {
			i.logger.Error(ctx, "could not BLPop from redis: %w", err)
			i.stopped.Signal()

			return err
		}

		if len(rawMessage) == 0 {
			continue
		}

		msg := Message{}
		err = json.Unmarshal([]byte(rawMessage[1]), &msg)
		if err != nil {
			i.logger.Error(ctx, "could not unmarshal message: %w", err)

			continue
		}

		messages <- &msg
		i.writeListReadMetric(ctx)
	}
}

func (i *redisListInput) Stop(_ context.Context) {
	i.stopped.Signal()
}

func (i *redisListInput) IsHealthy() bool {
	return i.healthCheckTimer.IsHealthy()
}

func (i *redisListInput) runMetricLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		i.writeListLengthMetric(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (i *redisListInput) writeListLengthMetric(ctx context.Context) {
	llen, err := i.client.LLen(ctx, i.settings.Key)
	if err != nil {
		i.logger.Error(ctx, "can not publish stream list metric data: %w", err)

		return
	}

	data := metric.Data{{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameRedisListInputLength,
		Dimensions: map[string]string{
			"StreamName": fmt.Sprintf("%s-%s", i.settings.ServerName, i.settings.Key),
		},
		Unit:  metric.UnitCountAverage,
		Value: float64(llen),
	}}

	i.mw.Write(ctx, data)
}

func (i *redisListInput) writeListReadMetric(ctx context.Context) {
	data := metric.Data{{
		MetricName: metricNameRedisListInputReads,
		Dimensions: map[string]string{
			"StreamName": fmt.Sprintf("%s-%s", i.settings.ServerName, i.settings.Key),
		},
		Value: 1.0,
	}}

	i.mw.Write(ctx, data)
}

func getRedisListInputDefaultMetrics(settings *RedisListInputSettings) metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameRedisListInputReads,
			Dimensions: map[string]string{
				"StreamName": fmt.Sprintf("%s-%s", settings.ServerName, settings.Key),
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
