package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type SnsConsumerConfig struct {
	ConsumerId string `cfg:"consumer_id"`
	WaitTime   int    `cfg:"consumer_wait_time"`
}

type SnsConsumerTarget struct {
	Family      string `mapstructure:"family"`
	Application string `mapstructure:"application"`
	TopicId     string `mapstructure:"topic_id"`
}

type snsConsumer struct {
	baseConsumer
}

func NewSnsConsumer(callback ConsumerCallback) *snsConsumer {
	return &snsConsumer{
		baseConsumer{
			callback: callback,
		},
	}
}

func (c *snsConsumer) Boot(config cfg.Config, logger mon.Logger) error {
	err := c.baseConsumer.Boot(config, logger)

	if err != nil {
		return err
	}

	cc := &SnsConsumerConfig{}
	config.Bind(cc)

	settings := SnsInputSettings{
		QueueId:  cc.ConsumerId,
		WaitTime: int64(cc.WaitTime),
	}

	targetConfig := make([]SnsConsumerTarget, 0)
	config.Unmarshal("consumer_targets", &targetConfig)

	targets := make([]SnsInputTarget, len(targetConfig))
	for i := range targetConfig {
		targets[i].Family = targetConfig[i].Family
		targets[i].Application = targetConfig[i].Application
		targets[i].TopicId = targetConfig[i].TopicId
	}

	c.input = NewSnsInput(config, logger, settings, targets)

	appId := cfg.AppId{}
	appId.PadFromConfig(config)
	c.name = fmt.Sprintf("consumer-%v-%v-%v", appId.Family, appId.Application, cc.ConsumerId)

	return nil
}
