package stream

import (
	"encoding/json"
	"errors"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/redis"
	"time"
)

const (
	metricNameRedisListInputLength = "StreamRedisListInputLength"
	metricNameRedisListInputReads  = "StreamRedisListInputReads"
)

type RedisListInputSettings struct {
	cfg.AppId
	ServerName string
	Key        string
	WaitTime   time.Duration
}

type redisListInput struct {
	logger   mon.Logger
	mw       mon.MetricWriter
	client   redis.Client
	settings *RedisListInputSettings

	channel           chan *Message
	stopped           bool
	fullyQualifiedKey string
}

func NewRedisListInput(config cfg.Config, logger mon.Logger, settings *RedisListInputSettings) Input {
	settings.PadFromConfig(config)
	client := redis.GetClient(config, logger, settings.ServerName)

	defaultMetrics := getRedisListInputDefaultMetrics(settings.AppId, settings.Key)
	mw := mon.NewMetricDaemonWriter(defaultMetrics...)

	return NewRedisListInputWithInterfaces(logger, client, mw, settings)
}

func NewRedisListInputWithInterfaces(logger mon.Logger, client redis.Client, mw mon.MetricWriter, settings *RedisListInputSettings) Input {
	fullyQualifiedKey := redis.GetFullyQualifiedKey(settings.AppId, settings.Key)

	return &redisListInput{
		logger:            logger,
		client:            client,
		settings:          settings,
		mw:                mw,
		channel:           make(chan *Message),
		fullyQualifiedKey: fullyQualifiedKey,
	}
}

func (i *redisListInput) Data() chan *Message {
	return i.channel
}

func (i *redisListInput) Run() error {
	defer close(i.channel)

	if i.settings.WaitTime == 0 {
		return errors.New("wait time should be bigger than 0")
	}

	go i.runMetricLoop()

	for {
		if i.stopped {
			return nil
		}

		rawMessage, err := i.client.BLPop(i.settings.WaitTime, i.fullyQualifiedKey)

		if err != nil && err.Error() != redis.Nil.Error() {
			i.logger.Error(err, "could not BLPop from redis")
			i.stopped = true
			return err
		}

		if len(rawMessage) == 0 {
			continue
		}

		msg := Message{}
		err = json.Unmarshal([]byte(rawMessage[1]), &msg)

		if err != nil {
			i.logger.Error(err, "could not unmarshal message")
			continue
		}

		i.channel <- &msg
		i.writeListReadMetric()
	}
}

func (i *redisListInput) Stop() {
	i.stopped = true
}

func (i *redisListInput) runMetricLoop() {
	ticker := time.NewTicker(1 * time.Second)

	for {
		i.writeListLengthMetric()
		<-ticker.C
	}
}

func (i *redisListInput) writeListLengthMetric() {
	llen, err := i.client.LLen(i.fullyQualifiedKey)

	if err != nil {
		i.logger.Error(err, "can not publish stream list metric data")
		return
	}

	data := mon.MetricData{{
		Priority:   mon.PriorityHigh,
		MetricName: metricNameRedisListInputLength,
		Dimensions: map[string]string{
			"StreamName": i.fullyQualifiedKey,
		},
		Unit:  mon.UnitCountAverage,
		Value: float64(llen),
	}}

	i.mw.Write(data)
}

func (i *redisListInput) writeListReadMetric() {
	data := mon.MetricData{{
		MetricName: metricNameRedisListInputReads,
		Dimensions: map[string]string{
			"StreamName": i.fullyQualifiedKey,
		},
		Value: 1.0,
	}}

	i.mw.Write(data)
}

func getRedisListInputDefaultMetrics(appId cfg.AppId, key string) mon.MetricData {
	fullyQualifiedKey := redis.GetFullyQualifiedKey(appId, key)

	return mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameRedisListInputReads,
			Dimensions: map[string]string{
				"StreamName": fullyQualifiedKey,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
	}
}
